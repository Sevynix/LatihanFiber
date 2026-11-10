package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"tugas2/app/model"
)

var ErrProjectLimit = errors.New("jumlah proyek sudah mencapai batas")

const MaxProjectsPerPortfolio = 20

type PortfolioRepository interface {
	EnsureIndexes(ctx context.Context) error
	FindByStudent(ctx context.Context, studentID int) (model.Portfolio, error)
	Upsert(ctx context.Context, studentID int, p model.Portfolio) (model.Portfolio, bool, error)
	AddProject(ctx context.Context, studentID int, p model.Project) (model.Portfolio, error)
	RemoveProject(ctx context.Context, studentID int, projectID string) error
	TopTechnologies(ctx context.Context, limit int) ([]model.TechnologyCount, error)
}

type portfolioDocument struct {
	ID           bson.ObjectID         `bson:"_id,omitempty"`
	StudentID    int                   `bson:"student_id"`
	Headline     string                `bson:"headline"`
	Bio          string                `bson:"bio"`
	Projects     []projectDocument     `bson:"projects"`
	Certificates []certificateDocument `bson:"certificates"`
	CreatedAt    time.Time             `bson:"created_at"`
	UpdatedAt    time.Time             `bson:"updated_at"`
}

type projectDocument struct {
	ID           bson.ObjectID `bson:"_id"`
	Title        string        `bson:"title"`
	Technologies []string      `bson:"technologies"`
	URL          string        `bson:"url"`
	Year         int           `bson:"year"`
}

type certificateDocument struct {
	Name   string    `bson:"name"`
	Issuer string    `bson:"issuer"`
	Date   time.Time `bson:"date"`
}

func newProjectDocument(p model.Project) projectDocument {
	techs := p.Technologies
	if techs == nil {
		techs = []string{}
	}
	return projectDocument{
		ID: bson.NewObjectID(), Title: p.Title, Technologies: techs,
		URL: p.URL, Year: p.Year,
	}
}

func (d portfolioDocument) toModel() model.Portfolio {
	projects := make([]model.Project, 0, len(d.Projects))
	for _, p := range d.Projects {
		techs := p.Technologies
		if techs == nil {
			techs = []string{}
		}
		projects = append(projects, model.Project{
			ID: p.ID.Hex(), Title: p.Title, Technologies: techs,
			URL: p.URL, Year: p.Year,
		})
	}
	certs := make([]model.Certificate, 0, len(d.Certificates))
	for _, c := range d.Certificates {
		certs = append(certs, model.Certificate{Name: c.Name, Issuer: c.Issuer, Date: c.Date})
	}
	return model.Portfolio{
		StudentID: d.StudentID, Headline: d.Headline, Bio: d.Bio,
		Projects: projects, Certificates: certs,
		CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
	}
}

type portfolioMongoRepository struct{ coll *mongo.Collection }

func NewPortfolioRepository(db *mongo.Database) PortfolioRepository {
	return &portfolioMongoRepository{coll: db.Collection("student_portfolios")}
}

func (r *portfolioMongoRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.coll.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "student_id", Value: 1}},
		Options: options.Index().SetUnique(true).SetName("student_unique"),
	})
	if err != nil {
		return fmt.Errorf("membuat index student_portfolios: %w", err)
	}
	return nil
}

func (r *portfolioMongoRepository) FindByStudent(ctx context.Context, studentID int) (model.Portfolio, error) {
	var doc portfolioDocument
	if err := r.coll.FindOne(ctx, bson.M{"student_id": studentID}).Decode(&doc); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return model.Portfolio{}, ErrNotFound
		}
		return model.Portfolio{}, fmt.Errorf("mengambil portofolio: %w", err)
	}
	return doc.toModel(), nil
}

func (r *portfolioMongoRepository) Upsert(ctx context.Context, studentID int, p model.Portfolio) (model.Portfolio, bool, error) {
	t := now()

	projects := make([]projectDocument, 0, len(p.Projects))
	for _, pr := range p.Projects {
		projects = append(projects, newProjectDocument(pr))
	}
	certs := make([]certificateDocument, 0, len(p.Certificates))
	for _, c := range p.Certificates {
		certs = append(certs, certificateDocument{Name: c.Name, Issuer: c.Issuer, Date: c.Date})
	}

	update := bson.M{
		"$set": bson.M{
			"headline": p.Headline, "bio": p.Bio,
			"projects": projects, "certificates": certs,
			"updated_at": t,
		},
		"$setOnInsert": bson.M{"student_id": studentID, "created_at": t},
	}

	var (
		res *mongo.UpdateResult
		err error
	)
	for attempt := 0; attempt < 2; attempt++ {
		res, err = r.coll.UpdateOne(ctx, bson.M{"student_id": studentID}, update,
			options.UpdateOne().SetUpsert(true))
		if err == nil || !mongo.IsDuplicateKeyError(err) {
			break
		}
	}
	if err != nil {
		return model.Portfolio{}, false, fmt.Errorf("menyimpan portofolio: %w", err)
	}

	saved, err := r.FindByStudent(ctx, studentID)
	if err != nil {
		return model.Portfolio{}, false, err
	}
	return saved, res.UpsertedCount > 0, nil
}

func (r *portfolioMongoRepository) AddProject(ctx context.Context, studentID int, p model.Project) (model.Portfolio, error) {
	limitField := "projects." + strconv.Itoa(MaxProjectsPerPortfolio-1)

	var updated portfolioDocument
	err := r.coll.FindOneAndUpdate(ctx,
		bson.M{"student_id": studentID, limitField: bson.M{"$exists": false}},
		bson.M{
			"$push": bson.M{"projects": newProjectDocument(p)},
			"$set":  bson.M{"updated_at": now()},
		},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&updated)
	if err == nil {
		return updated.toModel(), nil
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return model.Portfolio{}, fmt.Errorf("menambah proyek: %w", err)
	}

	count, err := r.coll.CountDocuments(ctx, bson.M{"student_id": studentID})
	if err != nil {
		return model.Portfolio{}, fmt.Errorf("memeriksa portofolio: %w", err)
	}
	if count == 0 {
		return model.Portfolio{}, ErrNotFound
	}
	return model.Portfolio{}, ErrProjectLimit
}

func (r *portfolioMongoRepository) RemoveProject(ctx context.Context, studentID int, projectID string) error {
	pid, err := parseID(projectID)
	if err != nil {
		return err
	}

	res, err := r.coll.UpdateOne(ctx,
		bson.M{"student_id": studentID, "projects._id": pid},
		bson.M{
			"$pull": bson.M{"projects": bson.M{"_id": pid}},
			"$set":  bson.M{"updated_at": now()},
		},
	)
	if err != nil {
		return fmt.Errorf("menghapus proyek: %w", err)
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *portfolioMongoRepository) TopTechnologies(ctx context.Context, limit int) ([]model.TechnologyCount, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$unwind", Value: "$projects"}},
		{{Key: "$unwind", Value: "$projects.technologies"}},
		{{Key: "$group", Value: bson.M{
			"_id":   "$projects.technologies",
			"total": bson.M{"$sum": 1},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "total", Value: -1}, {Key: "_id", Value: 1}}}},
		{{Key: "$limit", Value: int64(limit)}},
	}

	cursor, err := r.coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("menghitung teknologi: %w", err)
	}
	defer cursor.Close(ctx)

	var rows []struct {
		Technology string `bson:"_id"`
		Total      int    `bson:"total"`
	}
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, fmt.Errorf("membaca statistik teknologi: %w", err)
	}

	result := make([]model.TechnologyCount, 0, len(rows))
	for _, row := range rows {
		result = append(result, model.TechnologyCount{Technology: row.Technology, Total: row.Total})
	}
	return result, nil
}

package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"tugas2/app/model"
	"tugas2/app/repository"
	"tugas2/helper"
)

type PortfolioService struct {
	portfolios repository.PortfolioRepository
	students   repository.StudentRepository
	perms      *helper.PermissionSet
}

func NewPortfolioService(
	portfolios repository.PortfolioRepository,
	students repository.StudentRepository,
	perms *helper.PermissionSet,
) *PortfolioService {
	return &PortfolioService{portfolios: portfolios, students: students, perms: perms}
}

func (s *PortfolioService) authorize(ctx context.Context, c *fiber.Ctx, anyPermission string) (int, error) {
	current, ok := helper.CurrentUser(c)
	if !ok {
		return 0, helper.Unauthorized("belum terautentikasi")
	}

	studentID, valid := helper.ParamID(c)
	if !valid {
		return 0, helper.BadRequest("id harus berupa angka positif")
	}

	student, err := s.students.FindByID(ctx, studentID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			if !s.perms.Can(current.Role, anyPermission) {
				return 0, helper.Forbidden("tidak berhak mengakses portofolio ini")
			}
			return 0, helper.NotFound("student tidak ditemukan")
		}
		return 0, helper.Internal(err)
	}

	if !CanAccessStudent(current, ownerIDOf(student), s.perms, anyPermission) {
		return 0, helper.Forbidden("tidak berhak mengakses portofolio ini")
	}
	return studentID, nil
}

func (s *PortfolioService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	studentID, err := s.authorize(ctx, c, "student:read:any")
	if err != nil {
		return err
	}

	p, err := s.portfolios.FindByStudent(ctx, studentID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("portofolio belum dibuat")
		}
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusOK, "portofolio ditemukan", p)
}

func (s *PortfolioService) Put(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	studentID, err := s.authorize(ctx, c, "student:update:any")
	if err != nil {
		return err
	}

	var req model.PutPortfolioRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	req.Headline = strings.TrimSpace(req.Headline)
	req.Bio = strings.TrimSpace(req.Bio)
	for i := range req.Projects {
		normalizeProjectRequest(&req.Projects[i])
	}

	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	p := model.Portfolio{
		Headline:     req.Headline,
		Bio:          req.Bio,
		Projects:     make([]model.Project, 0, len(req.Projects)),
		Certificates: make([]model.Certificate, 0, len(req.Certificates)),
	}
	for _, pr := range req.Projects {
		p.Projects = append(p.Projects, toProject(pr))
	}
	for _, cr := range req.Certificates {
		date, err := time.Parse("2006-01-02", cr.Date)
		if err != nil {
			return helper.BadRequest("format tanggal sertifikat harus YYYY-MM-DD")
		}
		p.Certificates = append(p.Certificates, model.Certificate{
			Name: strings.TrimSpace(cr.Name), Issuer: strings.TrimSpace(cr.Issuer), Date: date,
		})
	}

	saved, created, err := s.portfolios.Upsert(ctx, studentID, p)
	if err != nil {
		return translatePortfolioError(err)
	}
	if created {
		return helper.Success(c, fiber.StatusCreated, "portofolio berhasil dibuat", saved)
	}
	return helper.Success(c, fiber.StatusOK, "portofolio berhasil diganti", saved)
}

func (s *PortfolioService) AddProject(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	studentID, err := s.authorize(ctx, c, "student:update:any")
	if err != nil {
		return err
	}

	var req model.ProjectRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	normalizeProjectRequest(&req)

	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	updated, err := s.portfolios.AddProject(ctx, studentID, toProject(req))
	if err != nil {
		return translatePortfolioError(err)
	}
	return helper.Success(c, fiber.StatusCreated, "proyek berhasil ditambahkan", updated)
}

func (s *PortfolioService) RemoveProject(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	studentID, err := s.authorize(ctx, c, "student:update:any")
	if err != nil {
		return err
	}

	if err := s.portfolios.RemoveProject(ctx, studentID, c.Params("projectId")); err != nil {
		return translatePortfolioError(err)
	}
	return helper.NoContent(c)
}

func (s *PortfolioService) TopTechnologies(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	top, err := s.portfolios.TopTechnologies(ctx, 10)
	if err != nil {
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusOK, "statistik teknologi berhasil diambil", top)
}

func normalizeProjectRequest(p *model.ProjectRequest) {
	p.Title = strings.TrimSpace(p.Title)
	p.URL = strings.TrimSpace(p.URL)
	p.Technologies = NormalizeTags(p.Technologies)
}

func toProject(p model.ProjectRequest) model.Project {
	return model.Project{Title: p.Title, Technologies: p.Technologies, URL: p.URL, Year: p.Year}
}

func translatePortfolioError(err error) error {
	switch {
	case errors.Is(err, repository.ErrInvalidID):
		return helper.BadRequest("id proyek tidak valid")
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound("portofolio atau proyek tidak ditemukan")
	case errors.Is(err, repository.ErrProjectLimit):
		return helper.NewError(fiber.StatusConflict, "PROJECT_LIMIT", "jumlah proyek sudah mencapai batas")
	default:
		return helper.Internal(err)
	}
}

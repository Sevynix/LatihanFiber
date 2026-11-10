package service_test

import (
	"context"
	"strconv"
	"sync"
	"time"

	"tugas2/app/model"
	"tugas2/app/repository"
)

var (
	_ repository.ReviewRepository = (*fakeReviewRepo)(nil)
	_ repository.CourseRepository = (*fakeCourseRepo)(nil)
)

type fakeReviewRepo struct {
	repository.ReviewRepository

	mu      sync.Mutex
	reviews []model.Review
}

func newFakeReviewRepo() *fakeReviewRepo {
	return &fakeReviewRepo{}
}

func (f *fakeReviewRepo) Create(_ context.Context, r model.Review) (model.Review, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, existing := range f.reviews {
		if existing.CourseID == r.CourseID && existing.UserID == r.UserID {
			return model.Review{}, repository.ErrDuplicate
		}
	}

	r.ID = strconv.Itoa(len(f.reviews) + 1)
	r.CreatedAt = time.Now()
	r.UpdatedAt = r.CreatedAt
	if r.Tags == nil {
		r.Tags = []string{}
	}
	r.Replies = []model.Reply{}

	f.reviews = append(f.reviews, r)
	return r, nil
}

func (f *fakeReviewRepo) FindByCourse(_ context.Context, q model.ReviewQuery) ([]model.Review, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	result := []model.Review{}
	for i := len(f.reviews) - 1; i >= 0; i-- {
		if f.reviews[i].CourseID != q.CourseID {
			continue
		}
		result = append(result, f.reviews[i])
		if q.Limit > 0 && len(result) >= q.Limit {
			break
		}
	}
	return result, nil
}

type fakeCourseRepo struct {
	repository.CourseRepository

	courses  map[int]model.Course
	enrolled map[[2]int]bool
}

func (f *fakeCourseRepo) FindByID(_ context.Context, id int) (model.Course, error) {
	c, ok := f.courses[id]
	if !ok {
		return model.Course{}, repository.ErrNotFound
	}
	return c, nil
}

func (f *fakeCourseRepo) IsEnrolled(_ context.Context, courseID, userID int) (bool, error) {
	return f.enrolled[[2]int{courseID, userID}], nil
}

package service

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"tugas2/app/model"
	"tugas2/app/repository"
	"tugas2/helper"
)

type StudentService struct {
	repo  repository.StudentRepository
	perms *helper.PermissionSet
}

func NewStudentService(repo repository.StudentRepository, perms *helper.PermissionSet) *StudentService {
	return &StudentService{repo: repo, perms: perms}
}

func translateError(err error) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound("student tidak ditemukan")
	case errors.Is(err, repository.ErrDuplicate):
		return helper.Conflict("NIM sudah terdaftar")
	default:
		return helper.Internal(err)
	}
}

func (s *StudentService) loadAuthorized(
	c *fiber.Ctx, ctx context.Context, id int, anyPermission string,
) (student model.Student, allowed bool, err error) {
	current, ok := helper.CurrentUser(c)
	if !ok {
		return model.Student{}, false, helper.Unauthorized("belum terautentikasi")
	}

	student, findErr := s.repo.FindByID(ctx, id)
	if findErr != nil {
		if errors.Is(findErr, repository.ErrNotFound) && !s.perms.Can(current.Role, anyPermission) {
			return model.Student{}, false, helper.Forbidden("tidak berhak mengakses data student ini")
		}
		return model.Student{}, false, translateError(findErr)
	}

	if !CanAccessStudent(current, ownerIDOf(student), s.perms, anyPermission) {
		return model.Student{}, false, helper.Forbidden("tidak berhak mengakses data student ini")
	}
	return student, true, nil
}

func (s *StudentService) List(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	q := helper.ParseListQuery(c)
	students, total, err := s.repo.FindAll(ctx, q)
	if err != nil {
		return helper.Internal(err)
	}
	return helper.SuccessList(c, "daftar student berhasil diambil", students, &model.Meta{
		Page: q.Page, Limit: q.Limit, Total: total,
		TotalPages: CountTotalPages(total, q.Limit),
	})
}

func (s *StudentService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}
	student, allowed, err := s.loadAuthorized(c, ctx, id, "student:read:any")
	if !allowed {
		return err
	}
	return helper.Success(c, fiber.StatusOK, "student ditemukan", student)
}

func (s *StudentService) Create(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	var req model.CreateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	req.Name = strings.TrimSpace(req.Name)
	req.NIM = strings.TrimSpace(req.NIM)

	if errs := ValidateCreate(req); len(errs) > 0 {
		return helper.Validation(errs)
	}

	ownerID := current.UserID

	newStudent, err := s.repo.Create(ctx, model.Student{
		NIM: req.NIM, Name: req.Name, Grade: req.Grade, IsActive: true,
		OwnerID: &ownerID,
	})
	if err != nil {
		return translateError(err)
	}
	return helper.Created(c, "student berhasil dibuat", newStudent,
		"/api/v1/students/"+strconv.Itoa(newStudent.ID))
}

func (s *StudentService) Replace(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}
	if _, allowed, err := s.loadAuthorized(c, ctx, id, "student:update:any"); !allowed {
		return err
	}

	var req model.ReplaceStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if errs := ValidateReplace(req); len(errs) > 0 {
		return helper.Validation(errs)
	}

	result, err := s.repo.Update(ctx, model.Student{
		ID: id, Name: req.Name, Grade: req.Grade, IsActive: req.IsActive,
	})
	if err != nil {
		return translateError(err)
	}
	return helper.Success(c, fiber.StatusOK, "student berhasil diganti seluruhnya", result)
}

func (s *StudentService) Patch(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}
	existing, allowed, authErr := s.loadAuthorized(c, ctx, id, "student:update:any")
	if !allowed {
		return authErr
	}

	var req model.PatchStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if IsEmptyPatch(req) {
		return helper.BadRequest("tidak ada field yang diubah")
	}

	updated, errs := ApplyPatch(existing, req)
	if len(errs) > 0 {
		return helper.Validation(errs)
	}

	result, err := s.repo.Update(ctx, updated)
	if err != nil {
		return translateError(err)
	}
	return helper.Success(c, fiber.StatusOK, "student berhasil diperbarui sebagian", result)
}

func (s *StudentService) Delete(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return translateError(err)
	}
	return helper.NoContent(c)
}
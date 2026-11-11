package service

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"

	"tugas2/app/model"
	"tugas2/app/repository"
	"tugas2/helper"
)

type UserService struct {
	repo  repository.UserRepository
	perms *helper.PermissionSet
}

func NewUserService(repo repository.UserRepository, perms *helper.PermissionSet) *UserService {
	return &UserService{repo: repo, perms: perms}
}

func translateUserError(err error, entity string) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound(entity + " tidak ditemukan")
	case errors.Is(err, repository.ErrDuplicate):
		return helper.Conflict("email sudah dipakai")
	default:
		return helper.Internal(err)
	}
}

// List godoc
//
// @Summary Daftar user dengan cursor pagination (admin atau staff)
// @Description Membutuhkan permission user:list. Header Accept: text/csv menghasilkan file CSV
// @Tags users
// @Produce json
// @Param limit query int false "Jumlah per halaman" default(10)
// @Param search query string false "Cari berdasarkan username atau email"
// @Param is_active query bool false "Saring berdasarkan status aktif"
// @Param cursor query string false "Cursor dari meta.next_cursor"
// @Success 200 {object} model.UserPage
// @Failure 400 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 406 {object} model.ErrorResponse "Format Accept tidak tersedia"
// @Security BearerAuth
// @Router /users [get]
func (s *UserService) List(c *fiber.Ctx) error {
	format, err := helper.Negotiate(c, helper.FormatJSON, helper.FormatCSV)
	if err != nil {
		return err
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	q, err := helper.ParseCursorQuery(c)
	if err != nil {
		return err
	}

	rows, err := s.repo.FindAfterCursor(ctx, q)
	if err != nil {
		return helper.Internal(err)
	}

	hasMore := len(rows) > q.Limit
	if hasMore {
		rows = rows[:q.Limit]
	}

	if format == helper.FormatCSV {
		return helper.WriteUsersCSV(c, rows)
	}

	meta := &model.CursorMeta{Limit: q.Limit, HasMore: hasMore}
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		meta.NextCursor = helper.EncodeCursor(last.CreatedAt, last.ID)
	}
	return c.JSON(model.UserPage{
		Success: true,
		Message: "daftar user berhasil diambil",
		Data:    rows,
		Meta:    *meta,
	})
}

// Get godoc
//
// @Summary Detail satu user
// @Description Pemilik akun, atau pemegang permission user:read:any
// @Tags users
// @Produce json
// @Param id path int true "ID user"
// @Success 200 {object} model.WebResponse{data=model.User}
// @Failure 400 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Security BearerAuth
// @Router /users/{id} [get]
func (s *UserService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}
	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	if !CanAccessUser(current, id, s.perms, "user:read:any") {
		return helper.Forbidden("tidak berhak mengakses data user lain")
	}

	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateUserError(err, "user")
	}
	return helper.Success(c, fiber.StatusOK, "user ditemukan", user)
}

// Patch godoc
//
// @Summary Mengubah email user
// @Description Pemilik akun, atau pemegang permission user:update:any.
// @Tags users
// @Accept json
// @Produce json
// @Param id path int true "ID user"
// @Param body body model.UpdateUserRequest true "Email baru"
// @Success 200 {object} model.WebResponse{data=model.User}
// @Failure 400 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 409 {object} model.ErrorResponse "Email sudah dipakai"
// @Failure 422 {object} model.ErrorResponse
// @Security BearerAuth
// @Router /users/{id} [patch]
func (s *UserService) Patch(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}
	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}
	if !CanAccessUser(current, id, s.perms, "user:update:any") {
		return helper.Forbidden("tidak berhak mengubah data user lain")
	}

	var req model.UpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if req.Email == nil {
		return helper.BadRequest("tidak ada field yang diubah")
	}
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}
	email := strings.TrimSpace(*req.Email)

	updated, err := s.repo.UpdateEmail(ctx, id, email)
	if err != nil {
		return translateUserError(err, "user")
	}
	return helper.Success(c, fiber.StatusOK, "user berhasil diperbarui", updated)
}

// AssignRole godoc
//
// @Summary Mengubah role user (admin)
// @Description Membutuhkan permission role:assign
// @Tags users
// @Accept json
// @Produce json
// @Param id path int true "ID user"
// @Param body body model.AssignRoleRequest true "Role baru"
// @Success 200 {object} model.WebResponse{data=model.User}
// @Failure 403 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 422 {object} model.ErrorResponse
// @Security BearerAuth
// @Router /users/{id}/role [patch]
func (s *UserService) AssignRole(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}
	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}
	var req model.AssignRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if errs := ValidateAssignRole(current, id, req, s.perms); len(errs) > 0 {
		return helper.Validation(errs)
	}

	result, err := s.repo.UpdateRole(ctx, id, strings.TrimSpace(req.Role))
	if err != nil {
		return translateUserError(err, "user")
	}
	return helper.Success(c, fiber.StatusOK, "role user berhasil diubah", result)
}

// Delete godoc
//
// @Summary Menghapus user (admin)
// @Description Membutuhkan permission user:delete. Tidak boleh menghapus akun sendiri
// @Tags users
// @Param id path int true "ID user"
// @Success 204 "Berhasil dihapus; tanpa body"
// @Failure 403 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Security BearerAuth
// @Router /users/{id} [delete]
func (s *UserService) Delete(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}
	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	if current.UserID == id {
		return helper.Forbidden("tidak boleh menghapus akun sendiri")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return translateUserError(err, "user")
	}
	return helper.NoContent(c)
}

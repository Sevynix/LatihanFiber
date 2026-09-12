package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"tugas2/app/model"
	"tugas2/app/repository"
	"tugas2/helper"
)

const refreshTokenBytes = 32

type AuthService struct {
	students   repository.StudentRepository
	tokens     repository.TokenRepository
	jwt        *helper.JWTManager
	refreshTTL time.Duration
}

func NewAuthService(
	students repository.StudentRepository,
	tokens repository.TokenRepository,
	jwtManager *helper.JWTManager,
	refreshTTL time.Duration,
) *AuthService {
	return &AuthService{
		students: students, tokens: tokens, jwt: jwtManager, refreshTTL: refreshTTL,
	}
}

func (s *AuthService) Register(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}
	req.Username = strings.TrimSpace(req.Username)
	req.Name = strings.TrimSpace(req.Name)
	req.NIM = strings.TrimSpace(req.NIM)

	if errs := ValidateRegister(req); len(errs) > 0 {
		return helper.FailValidation(c, errs)
	}

	hashed, err := helper.HashPassword(req.Password)
	if err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError, "gagal memproses password")
	}

	created, err := s.students.CreateWithCredentials(ctx, model.Student{
		NIM: req.NIM, Name: req.Name, Grade: 0, IsActive: true,
		Username: req.Username, PasswordHash: hashed, Role: "student",
	})
	if err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return helper.Fail(c, fiber.StatusConflict, "username atau NIM sudah dipakai")
		}
		return helper.Fail(c, fiber.StatusInternalServerError, "gagal mendaftarkan student")
	}
	return helper.Created(c, "pendaftaran berhasil", created,
		"/api/v1/students/"+strconv.Itoa(created.ID))
}

func (s *AuthService) Login(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}
	if errs := ValidateLogin(req); len(errs) > 0 {
		return helper.FailValidation(c, errs)
	}

	student, err := s.students.FindByUsername(ctx, strings.TrimSpace(req.Username))
	if err != nil {
		helper.VerifyDummyPassword(req.Password)
		return helper.Fail(c, fiber.StatusUnauthorized, "username atau password salah")
	}
	if !helper.VerifyPassword(student.PasswordHash, req.Password) {
		return helper.Fail(c, fiber.StatusUnauthorized, "username atau password salah")
	}
	if !student.IsActive {
		return helper.Fail(c, fiber.StatusForbidden, "akun dinonaktifkan")
	}

	pair, err := s.issueTokenPair(ctx, student)
	if err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError, "gagal membuat token")
	}
	return helper.Success(c, fiber.StatusOK, "login berhasil", pair)
}

func (s *AuthService) Refresh(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}
	if strings.TrimSpace(req.RefreshToken) == "" {
		return helper.Fail(c, fiber.StatusBadRequest, "refresh_token wajib diisi")
	}

	hash := helper.SHA256Hex(req.RefreshToken)
	stored, err := s.tokens.FindActive(ctx, hash)
	if err != nil {
		return helper.Fail(c, fiber.StatusUnauthorized,
			"refresh token tidak valid atau sudah kedaluwarsa")
	}

	student, err := s.students.FindByID(ctx, stored.StudentID)
	if err != nil || !student.IsActive {
		return helper.Fail(c, fiber.StatusUnauthorized, "akun tidak dapat dipakai")
	}

	if err := s.tokens.Revoke(ctx, hash); err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError, "gagal memperbarui token")
	}

	pair, err := s.issueTokenPair(ctx, student)
	if err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError, "gagal membuat token")
	}
	return helper.Success(c, fiber.StatusOK, "token berhasil diperbarui", pair)
}

func (s *AuthService) Logout(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}
	if strings.TrimSpace(req.RefreshToken) != "" {
		_ = s.tokens.Revoke(ctx, helper.SHA256Hex(req.RefreshToken))
	}
	return helper.Success(c, fiber.StatusOK, "logout berhasil", nil)
}

func (s *AuthService) Me(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	authStudent, ok := helper.CurrentStudent(c)
	if !ok {
		return helper.Fail(c, fiber.StatusUnauthorized, "belum terautentikasi")
	}
	student, err := s.students.FindByID(ctx, authStudent.StudentID)
	if err != nil {
		return helper.Fail(c, fiber.StatusUnauthorized, "student tidak ditemukan")
	}
	return helper.Success(c, fiber.StatusOK, "profil berhasil diambil", student)
}

func (s *AuthService) issueTokenPair(ctx context.Context, student model.Student) (model.TokenPair, error) {
	accessToken, err := s.jwt.GenerateAccess(student)
	if err != nil {
		return model.TokenPair{}, err
	}
	refreshToken, err := helper.RandomToken(refreshTokenBytes)
	if err != nil {
		return model.TokenPair{}, err
	}
	err = s.tokens.Save(ctx, model.RefreshToken{
		StudentID: student.ID,
		TokenHash: helper.SHA256Hex(refreshToken),
		ExpiresAt: time.Now().Add(s.refreshTTL),
	})
	if err != nil {
		return model.TokenPair{}, err
	}
	return model.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.jwt.AccessTTL().Seconds()),
	}, nil
}

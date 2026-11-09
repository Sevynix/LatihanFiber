package service

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"tugas2/app/model"
	"tugas2/app/repository"
	"tugas2/helper"
)

type SupervisionService struct {
	supervisors repository.SupervisorRepository
	tx          *repository.TxManager
}

func NewSupervisionService(supervisors repository.SupervisorRepository, tx *repository.TxManager) *SupervisionService {
	return &SupervisionService{supervisors: supervisors, tx: tx}
}

var errSameSupervisor = errors.New("mahasiswa sudah dibimbing pembimbing ini")

func (s *SupervisionService) List(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	list, err := s.supervisors.FindAll(ctx)
	if err != nil {
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusOK, "daftar pembimbing berhasil diambil", list)
}

func (s *SupervisionService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}
	supervisor, err := s.supervisors.FindByID(ctx, id)
	if err != nil {
		return translateSupervisionError(err)
	}
	return helper.Success(c, fiber.StatusOK, "pembimbing ditemukan", supervisor)
}

func (s *SupervisionService) Assign(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	supervisorID, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	var req model.AssignSupervisorRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	var assigned model.Supervisor
	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		repo := s.supervisors.WithTx(tx)

		if err := repo.LockStudent(ctx, req.StudentID); err != nil {
			return err
		}

		var err error
		assigned, err = repo.IncrementSupervised(ctx, supervisorID)
		if err != nil {
			return err
		}

		return repo.InsertSupervision(ctx, req.StudentID, supervisorID)
	})
	if err != nil {
		return translateSupervisionError(err)
	}
	return helper.Created(c, "pembimbing berhasil ditetapkan",
		fiber.Map{"student_id": req.StudentID, "supervisor": assigned},
		"/api/v1/supervisors/"+strconv.Itoa(supervisorID))
}

func (s *SupervisionService) Switch(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.SwitchSupervisorRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	var moved model.Supervisor
	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		repo := s.supervisors.WithTx(tx)

		if err := repo.LockStudent(ctx, req.StudentID); err != nil {
			return err
		}

		fromID, err := repo.FindSupervisorIDByStudent(ctx, req.StudentID)
		if err != nil {
			return err
		}
		if fromID == req.ToSupervisorID {
			return errSameSupervisor
		}

		if err := repo.LockSupervisors(ctx, fromID, req.ToSupervisorID); err != nil {
			return err
		}

		if _, err := repo.DecrementSupervised(ctx, fromID); err != nil {
			return err
		}
		moved, err = repo.IncrementSupervised(ctx, req.ToSupervisorID)
		if err != nil {
			return err
		}
		return repo.MoveSupervision(ctx, req.StudentID, req.ToSupervisorID)
	})
	if err != nil {
		return translateSupervisionError(err)
	}
	return helper.Success(c, fiber.StatusOK, "pembimbing berhasil diganti",
		fiber.Map{"student_id": req.StudentID, "supervisor": moved})
}

func (s *SupervisionService) UpdateQuota(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	var req model.UpdateSupervisorQuotaRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	updated, err := s.supervisors.UpdateQuota(ctx, id, req.Quota, req.Version)
	if err != nil {
		return translateSupervisionError(err)
	}
	return helper.Success(c, fiber.StatusOK, "kuota pembimbing berhasil diubah", updated)
}

func translateSupervisionError(err error) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound("pembimbing tidak ditemukan")
	case errors.Is(err, repository.ErrStudentNotFound):
		return helper.NotFound("mahasiswa tidak ditemukan")
	case errors.Is(err, repository.ErrSupervisorFull):
		return helper.NewError(fiber.StatusConflict, "SUPERVISOR_FULL", "kuota pembimbing sudah penuh")
	case errors.Is(err, repository.ErrDuplicate):
		return helper.NewError(fiber.StatusConflict, "ALREADY_SUPERVISED", "mahasiswa sudah memiliki pembimbing")
	case errors.Is(err, repository.ErrNotSupervised):
		return helper.NewError(fiber.StatusConflict, "NOT_SUPERVISED", "mahasiswa belum memiliki pembimbing")
	case errors.Is(err, errSameSupervisor):
		return helper.NewError(fiber.StatusConflict, "SAME_SUPERVISOR", "mahasiswa sudah dibimbing pembimbing ini")
	case errors.Is(err, repository.ErrSupervisorVersion):
		return helper.NewError(fiber.StatusConflict, "VERSION_CONFLICT",
			"data pembimbing sudah diubah pihak lain, muat ulang lalu coba lagi")
	case errors.Is(err, repository.ErrSupervisorQuotaBelow):
		return helper.NewError(fiber.StatusConflict, "QUOTA_BELOW_SUPERVISED",
			"kuota tidak boleh lebih kecil dari jumlah mahasiswa bimbingan saat ini")
	default:
		return helper.Internal(err)
	}
}

package main

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"tugas2/app/model"
)

var students []model.Student
var nextID = 1

func findStudentIndex(id int) int {
	for i := range students {
		if students[i].ID == id {
			return i
		}
	}
	return -1
}

func cocokPencarian(s model.Student, kata string) bool {
	kata = strings.ToLower(kata)
	return strings.Contains(strings.ToLower(s.Name), kata) ||
		strings.Contains(strings.ToLower(s.NIM), kata)
}

func paramID(c *fiber.Ctx) (int, bool) {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id < 1 {
		return 0, false
	}
	return id, true
}

func listStudents(c *fiber.Ctx) error {
	q := parseListQuery(c)

	hasil := []model.Student{}
	for _, s := range students {
		if q.IsActive != nil && s.IsActive != *q.IsActive {
			continue
		}
		if q.Search != "" && !cocokPencarian(s, q.Search) {
			continue
		}
		hasil = append(hasil, s)
	}

	sort.SliceStable(hasil, func(i, j int) bool {
		var lebihKecil bool
		switch q.Sort {
		case "name":
			lebihKecil = hasil[i].Name < hasil[j].Name
		case "nim":
			lebihKecil = hasil[i].NIM < hasil[j].NIM
		case "grade":
			lebihKecil = hasil[i].Grade < hasil[j].Grade
		case "created_at":
			lebihKecil = hasil[i].CreatedAt.Before(hasil[j].CreatedAt)
		default:
			lebihKecil = hasil[i].ID < hasil[j].ID
		}
		if q.Order == "desc" {
			return !lebihKecil
		}
		return lebihKecil
	})

	total := len(hasil)
	totalPages := (total + q.Limit - 1) / q.Limit
	mulai := (q.Page - 1) * q.Limit
	if mulai > total {
		mulai = total
	}
	akhir := mulai + q.Limit
	if akhir > total {
		akhir = total
	}

	return okList(c, "daftar student berhasil diambil", hasil[mulai:akhir], &model.Meta{
		Page: q.Page, Limit: q.Limit, Total: total, TotalPages: totalPages,
	})
}

func getStudent(c *fiber.Ctx) error {
	id, valid := paramID(c)
	if !valid {
		return fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}
	i := findStudentIndex(id)
	if i == -1 {
		return fail(c, fiber.StatusNotFound, "student tidak ditemukan")
	}
	return ok(c, "student ditemukan", students[i])
}

func createStudent(c *fiber.Ctx) error {
	var req model.CreateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}

	req.Name = strings.TrimSpace(req.Name)
	req.NIM = strings.TrimSpace(req.NIM)

	errs := map[string]string{}
	if req.Name == "" {
		errs["name"] = "wajib diisi"
	}
	if req.NIM == "" {
		errs["nim"] = "wajib diisi"
	}
	if req.Grade < 0 || req.Grade > 100 {
		errs["grade"] = "harus di antara 0 dan 100"
	}
	if len(errs) > 0 {
		return failValidation(c, errs)
	}

	for _, s := range students {
		if s.NIM == req.NIM {
			return fail(c, fiber.StatusConflict, "NIM sudah terdaftar")
		}
	}

	baru := model.Student{
		ID: nextID, NIM: req.NIM, Name: req.Name,
		Grade: req.Grade, IsActive: true, CreatedAt: time.Now(),
	}
	students = append(students, baru)
	nextID++

	return created(c, "student berhasil dibuat", baru, "/api/v1/students/"+strconv.Itoa(baru.ID))
}

func replaceStudent(c *fiber.Ctx) error {
	id, valid := paramID(c)
	if !valid {
		return fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}
	i := findStudentIndex(id)
	if i == -1 {
		return fail(c, fiber.StatusNotFound, "student tidak ditemukan")
	}

	var req model.ReplaceStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}

	errs := map[string]string{}
	if strings.TrimSpace(req.Name) == "" {
		errs["name"] = "wajib diisi pada PUT"
	}
	if req.Grade < 0 || req.Grade > 100 {
		errs["grade"] = "harus di antara 0 dan 100"
	}
	if len(errs) > 0 {
		return failValidation(c, errs)
	}

	students[i].Name = req.Name
	students[i].Grade = req.Grade
	students[i].IsActive = req.IsActive

	return ok(c, "student berhasil diganti seluruhnya", students[i])
}

func patchStudent(c *fiber.Ctx) error {
	id, valid := paramID(c)
	if !valid {
		return fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}
	i := findStudentIndex(id)
	if i == -1 {
		return fail(c, fiber.StatusNotFound, "student tidak ditemukan")
	}

	var req model.PatchStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}

	if req.Name == nil && req.Grade == nil && req.IsActive == nil {
		return fail(c, fiber.StatusBadRequest, "tidak ada field yang diubah")
	}

	if req.Name != nil {
		if strings.TrimSpace(*req.Name) == "" {
			return failValidation(c, map[string]string{"name": "tidak boleh kosong"})
		}
		students[i].Name = *req.Name
	}
	if req.Grade != nil {
		if *req.Grade < 0 || *req.Grade > 100 {
			return failValidation(c, map[string]string{"grade": "harus di antara 0 dan 100"})
		}
		students[i].Grade = *req.Grade
	}
	if req.IsActive != nil {
		students[i].IsActive = *req.IsActive
	}

	return ok(c, "student berhasil diperbarui sebagian", students[i])
}

func deleteStudent(c *fiber.Ctx) error {
	id, valid := paramID(c)
	if !valid {
		return fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}
	i := findStudentIndex(id)
	if i == -1 {
		return fail(c, fiber.StatusNotFound, "student tidak ditemukan")
	}

	students = append(students[:i], students[i+1:]...)

	return noContent(c)
}

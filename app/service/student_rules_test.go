package service

import (
	"testing"

	"tugas2/app/model"
)

func TestCountTotalPages(t *testing.T) {
	cases := []struct{ total, limit, want int }{
		{0, 10, 0},
		{11, 10, 2},
		{20, 10, 2},
	}
	for _, tc := range cases {
		if got := CountTotalPages(tc.total, tc.limit); got != tc.want {
			t.Errorf("total=%d limit=%d: harap %d, dapat %d", tc.total, tc.limit, tc.want, got)
		}
	}
}

func TestValidateCreate_KosongSemua(t *testing.T) {
	errs := ValidateCreate(model.CreateStudentRequest{})
	if len(errs) != 3 {
		t.Fatalf("harap 3 error (name, nim, grade), dapat %d: %v", len(errs), errs)
	}
}

func TestApplyPatch(t *testing.T) {
	initial := model.Student{ID: 1, Name: "Sari", NIM: "123", Grade: 80, IsActive: true}
	inactive := false
	result, errs := ApplyPatch(initial, model.PatchStudentRequest{IsActive: &inactive})
	if len(errs) != 0 {
		t.Fatalf("tidak seharusnya ada error: %v", errs)
	}
	if result.IsActive {
		t.Error("is_active seharusnya berubah menjadi false")
	}
	if result.Name != "Sari" {
		t.Error("field yang tidak dikirim seharusnya tidak berubah")
	}
}
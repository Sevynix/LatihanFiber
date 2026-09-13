package service

import (
	"testing"
	"tugas2/app/model"
)

func TestCheckPasswordStrength(t *testing.T) {
	cases := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"terlalu pendek", "abc123", true},
		{"tanpa angka", "passwordsaja", true},
		{"tanpa huruf", "12345678", true},
		{"password umum", "password123", true},
		{"valid", "rahasia123", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			msg := checkPasswordStrength(tc.password)
			gotErr := msg != ""
			if gotErr != tc.wantErr {
				t.Errorf("checkPasswordStrength(%q) = %q, wantErr=%v", tc.password, msg, tc.wantErr)
			}
		})
	}
}

func TestValidateRegister(t *testing.T) {
	req := model.RegisterRequest{
		Username: "sa",
		Password: "rahasia123",
		NIM:      "",
		Name:     "",
	}
	errs := ValidateRegister(req)

	if errs["username"] == "" {
		t.Error("username terlalu pendek seharusnya menghasilkan error")
	}
	if errs["nim"] == "" {
		t.Error("nim kosong seharusnya menghasilkan error")
	}
	if errs["name"] == "" {
		t.Error("name kosong seharusnya menghasilkan error")
	}
}

func TestValidateLogin(t *testing.T) {
	errs := ValidateLogin(model.LoginRequest{Username: "", Password: ""})
	if errs["username"] == "" || errs["password"] == "" {
		t.Error("username dan password kosong seharusnya menghasilkan error")
	}

	errsValid := ValidateLogin(model.LoginRequest{Username: "sari", Password: "rahasia123"})
	if len(errsValid) != 0 {
		t.Errorf("input valid seharusnya tidak menghasilkan error, dapat: %v", errsValid)
	}
}

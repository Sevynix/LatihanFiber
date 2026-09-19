package service

import (
	"strings"
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
		{"terlalu panjang untuk bcrypt", strings.Repeat("a1", 37), true},
		{"tepat 72 byte", strings.Repeat("a1", 36), false},
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

func TestValidateRegister_SemuaSalah(t *testing.T) {
	req := model.RegisterRequest{
		Username: "sa",
		Email:    "bukan-email",
		Password: "abc",
	}
	errs := ValidateRegister(req)

	for _, field := range []string{"username", "email", "password"} {
		if errs[field] == "" {
			t.Errorf("field %q seharusnya menghasilkan error", field)
		}
	}
}

func TestValidateRegister_Valid(t *testing.T) {
	req := model.RegisterRequest{
		Username: "sari.w_1",
		Email:    "sari@example.com",
		Password: "rahasia123",
	}
	if errs := ValidateRegister(req); len(errs) != 0 {
		t.Errorf("input valid seharusnya tidak menghasilkan error, dapat: %v", errs)
	}
}

func TestValidateRegister_BatasPanjang(t *testing.T) {
	req := model.RegisterRequest{
		Username: strings.Repeat("a", maxUsernameLength+1),
		Email:    strings.Repeat("a", maxEmailLength) + "@example.com",
		Password: "rahasia123",
	}
	errs := ValidateRegister(req)
	if errs["username"] == "" {
		t.Error("username melebihi kolom VARCHAR(50) seharusnya ditolak")
	}
	if errs["email"] == "" {
		t.Error("email melebihi kolom VARCHAR(100) seharusnya ditolak")
	}
}

func TestIsValidEmail(t *testing.T) {
	cases := []struct {
		email string
		want  bool
	}{
		{"sari@example.com", true},
		{"sari.w+tag@mail.unair.ac.id", true},
		{"", false},
		{"sari", false},
		{"sari@", false},
		{"@example.com", false},
		{"sari@localhost", false},
		{"Sari <sari@example.com>", false},
		{"sari@example.com, budi@example.com", false},
	}
	for _, tc := range cases {
		if got := isValidEmail(tc.email); got != tc.want {
			t.Errorf("isValidEmail(%q) = %v, harap %v", tc.email, got, tc.want)
		}
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
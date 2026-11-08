package service

import (
	"strings"
	"testing"

	"tugas2/app/model"
	"tugas2/helper"
)

func TestPasswordStrength(t *testing.T) {
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
			req := model.RegisterRequest{
				Username: "sari.w_1",
				Email:    "sari@example.com",
				Password: tc.password,
			}
			errs := helper.ValidateStruct(req)
			gotErr := errs["password"] != ""
			if gotErr != tc.wantErr {
				t.Errorf("password %q: error=%q, wantErr=%v", tc.password, errs["password"], tc.wantErr)
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
	errs := helper.ValidateStruct(req)

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
	if errs := helper.ValidateStruct(req); len(errs) != 0 {
		t.Errorf("input valid seharusnya tidak menghasilkan error, dapat: %v", errs)
	}
}

func TestValidateRegister_BatasPanjang(t *testing.T) {
	req := model.RegisterRequest{
		Username: strings.Repeat("a", 31),
		Email:    strings.Repeat("a", 121) + "@example.com",
		Password: "rahasia123",
	}
	errs := helper.ValidateStruct(req)
	if errs["username"] == "" {
		t.Error("username melebihi batas seharusnya ditolak")
	}
	if errs["email"] == "" {
		t.Error("email melebihi batas seharusnya ditolak")
	}
}

func TestEmailFormat(t *testing.T) {
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
	}
	for _, tc := range cases {
		req := model.RegisterRequest{
			Username: "sari.w_1",
			Email:    tc.email,
			Password: "rahasia123",
		}
		errs := helper.ValidateStruct(req)
		if got := errs["email"] == ""; got != tc.want {
			t.Errorf("email %q: valid=%v, harap %v", tc.email, got, tc.want)
		}
	}
}

func TestValidateLogin(t *testing.T) {
	errs := helper.ValidateStruct(model.LoginRequest{Username: "", Password: ""})
	if errs["username"] == "" || errs["password"] == "" {
		t.Error("username dan password kosong seharusnya menghasilkan error")
	}

	errsValid := helper.ValidateStruct(model.LoginRequest{Username: "sari", Password: "rahasia123"})
	if len(errsValid) != 0 {
		t.Errorf("input valid seharusnya tidak menghasilkan error, dapat: %v", errsValid)
	}
}
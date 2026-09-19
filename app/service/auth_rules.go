package service

import (
	"net/mail"
	"strings"
	"unicode"

	"tugas2/app/model"
)

const (
	minPasswordLength = 8
	maxPasswordLength = 72

	maxUsernameLength = 50
	maxEmailLength    = 100
)

func ValidateRegister(req model.RegisterRequest) map[string]string {
	errs := map[string]string{}

	username := strings.TrimSpace(req.Username)
	switch {
	case username == "":
		errs["username"] = "wajib diisi"
	case len(username) < 3:
		errs["username"] = "minimal 3 karakter"
	case len(username) > maxUsernameLength:
		errs["username"] = "maksimal 50 karakter"
	case !isValidUsername(username):
		errs["username"] = "hanya boleh huruf, angka, titik, dan garis bawah"
	}

	if !isValidEmail(req.Email) {
		errs["email"] = "format email tidak valid"
	}
	if msg := checkPasswordStrength(req.Password); msg != "" {
		errs["password"] = msg
	}
	return errs
}

func ValidateLogin(req model.LoginRequest) map[string]string {
	errs := map[string]string{}
	if strings.TrimSpace(req.Username) == "" {
		errs["username"] = "wajib diisi"
	}
	if req.Password == "" {
		errs["password"] = "wajib diisi"
	}
	return errs
}

func checkPasswordStrength(password string) string {
	if len(password) < minPasswordLength {
		return "minimal 8 karakter"
	}
	if len(password) > maxPasswordLength {
		return "maksimal 72 karakter"
	}
	var hasLetter, hasDigit bool
	for _, r := range password {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return "harus memuat huruf dan angka"
	}
	weak := map[string]bool{
		"password1": true, "12345678": true, "qwerty123": true,
		"admin123": true, "password123": true,
	}
	if weak[strings.ToLower(password)] {
		return "password terlalu umum"
	}
	return ""
}

func isValidUsername(username string) bool {
	for _, r := range username {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '.' && r != '_' {
			return false
		}
	}
	return true
}

func isValidEmail(email string) bool {
	if len(email) > maxEmailLength {
		return false
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return false
	}
	at := strings.LastIndex(email, "@")
	return strings.Contains(email[at+1:], ".")
}
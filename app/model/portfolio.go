package model

import "time"

type Portfolio struct {
	StudentID    int           `json:"student_id"`
	Headline     string        `json:"headline"`
	Bio          string        `json:"bio"`
	Projects     []Project     `json:"projects"`
	Certificates []Certificate `json:"certificates"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
}

type Project struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Technologies []string `json:"technologies"`
	URL          string   `json:"url"`
	Year         int      `json:"year"`
}

type Certificate struct {
	Name   string    `json:"name"`
	Issuer string    `json:"issuer"`
	Date   time.Time `json:"date"`
}

type ProjectRequest struct {
	Title        string   `json:"title" validate:"required,min=2,max=100"`
	Technologies []string `json:"technologies" validate:"required,min=1,max=15,dive,min=1,max=30"`
	URL          string   `json:"url" validate:"omitempty,url,max=300"`
	Year         int      `json:"year" validate:"required,min=2000,max=2100"`
}

type CertificateRequest struct {
	Name   string `json:"name" validate:"required,min=2,max=150"`
	Issuer string `json:"issuer" validate:"required,min=2,max=100"`
	Date   string `json:"date" validate:"required,datetime=2006-01-02"`
}

type PutPortfolioRequest struct {
	Headline     string               `json:"headline" validate:"required,min=3,max=100"`
	Bio          string               `json:"bio" validate:"max=1000"`
	Projects     []ProjectRequest     `json:"projects" validate:"max=20,dive"`
	Certificates []CertificateRequest `json:"certificates" validate:"max=50,dive"`
}

type TechnologyCount struct {
	Technology string `json:"technology"`
	Total      int    `json:"total"`
}

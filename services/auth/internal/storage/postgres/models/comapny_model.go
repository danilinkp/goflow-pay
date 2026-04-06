package models

import (
	"auth/internal/domain/entities"
	"time"

	"github.com/google/uuid"
)

type CompanyModel struct {
	CompanyId  uuid.UUID `db:"company_id"`
	Name       string    `db:"name"`
	InviteCode string    `db:"invite_code"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
}

func (m *CompanyModel) ToDomain() *entities.Company {
	return entities.ReconstructCompany(m.CompanyId, m.Name, m.InviteCode, m.CreatedAt, m.UpdatedAt)
}

func ToCompanyModel(in *entities.Company) *CompanyModel {
	return &CompanyModel{
		CompanyId:  in.CompanyId(),
		Name:       in.Name(),
		InviteCode: in.InviteCode(),
		CreatedAt:  in.CreatedAt(),
		UpdatedAt:  in.UpdatedAt(),
	}
}

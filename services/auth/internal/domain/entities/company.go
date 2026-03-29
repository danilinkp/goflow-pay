package entities

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Company struct {
	companyId  uuid.UUID
	name       string
	inviteCode string
	createdAt  time.Time
	updatedAt  time.Time
}

func NewCompany(name string) (*Company, error) {
	if name == "" {
		return nil, fmt.Errorf("company name cannot be empty")
	}

	now := time.Now().UTC()
	return &Company{
		companyId:  uuid.New(),
		name:       name,
		inviteCode: generateInviteCode(),
		createdAt:  now,
		updatedAt:  now,
	}, nil
}

func (c *Company) CompanyId() uuid.UUID { return c.companyId }
func (c *Company) Name() string         { return c.name }
func (c *Company) InviteCode() string   { return c.inviteCode }
func (c *Company) CreatedAt() time.Time { return c.createdAt }
func (c *Company) UpdatedAt() time.Time { return c.updatedAt }

func (c *Company) UpdateName(newName string) error {
	if newName == "" {
		return fmt.Errorf("%s: name cannot be empty", "update name")
	}
	c.name = newName
	c.updatedAt = time.Now().UTC()
	return nil
}

func generateInviteCode() string {
	return strings.ToUpper(uuid.New().String()[:8])
}

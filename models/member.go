package models

import (
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/Arup3201/torb/core"
)

/*
Gorm Custom Data Type: UserRole

Implements the sql.Scanner and sql.Valuer for gorm to recieve and save it
into the database.

It does not implement GormDBDataTypeInterface/GormDataTypeInterface, so it
takes the first field (Role) type (string) as the data type.
*/
type UserRole struct {
	String string
}

// Extracts the role from UserRole
// Returns error if role is not a valid string
func (r *UserRole) Scan(value any) error {
	role, ok := value.(string)
	if !ok {
		return fmt.Errorf("Failed to extract role value %v as string", value)
	}

	r.String = role
	return nil
}

// Returns the role value of the UserRole
// Returns error if role is not Owner/Member
func (r UserRole) Value() (driver.Value, error) {
	if r.String != core.ROLE_OWNER && r.String != core.ROLE_MEMBER {
		return nil, fmt.Errorf("Invalid role %s", r.String)
	}

	return r.String, nil
}

type Member struct {
	ProjectID string `gorm:"primaryKey"`
	UserID    string `gorm:"primaryKey"`
	Role      UserRole
	CreatedAt time.Time
	UpdatedAt time.Time
}

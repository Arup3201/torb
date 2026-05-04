package models

import (
	"database/sql/driver"
	"fmt"
	"slices"
	"time"

	"github.com/Arup3201/torb/core"
)

/*
Gorm Custom Data Type: JoinStatus

Implements the sql.Scanner and sql.Valuer for gorm to recieve and save it
into the database.

It does not implement GormDBDataTypeInterface/GormDataTypeInterface, so it
takes the first field (Role) type (string) as the data type.
*/
type JoinStatus struct {
	String string
}

// Extracts the status from JoinStatus
// Returns error if status is not a valid string
func (r *JoinStatus) Scan(value any) error {
	status, ok := value.(string)
	if !ok {
		return fmt.Errorf("Failed to extract join status value %v as string", value)
	}

	r.String = status
	return nil
}

// Returns the status value of the JoinStatus
// Returns error if status is not Pending/Accepted/Rejected
func (r JoinStatus) Value() (driver.Value, error) {
	if !slices.Contains([]string{
		core.JOIN_STATUS_PENDING,
		core.JOIN_STATUS_ACCEPTED,
		core.JOIN_STATUS_REJECTED,
	}, r.String) {
		return nil, fmt.Errorf("Invalid join status %s", r.String)
	}

	return r.String, nil
}

type JoinRequest struct {
	ProjectID string `gorm:"primaryKey"`
	UserID    string `gorm:"primaryKey"`
	Status    JoinStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

package models

import (
	"database/sql/driver"
	"fmt"
	"slices"
	"time"

	"github.com/Arup3201/torb/core"
)

/*
Gorm Custom Data Type: TaskStatus

Implements the sql.Scanner and sql.Valuer for gorm to recieve and save it
into the database.

It does not implement GormDBDataTypeInterface/GormDataTypeInterface, so it
takes the first field (Role) type (string) as the data type.
*/
type TaskStatus struct {
	String string
}

// Extracts the status from TaskStatus
// Returns error if status is not a valid string
func (r *TaskStatus) Scan(value any) error {
	status, ok := value.(string)
	if !ok {
		return fmt.Errorf("Failed to extract task status value %v as string", value)
	}

	r.String = status
	return nil
}

// Returns the status value of the TaskStatus
// Returns error if status is not Pending/Accepted/Rejected
func (r TaskStatus) Value() (driver.Value, error) {
	if !slices.Contains([]string{
		core.TASK_STATUS_UNASSIGNED,
		core.TASK_STATUS_ONGOING,
		core.TASK_STATUS_COMPLETED,
		core.TASK_STATUS_ABANDONED,
	}, r.String) {
		return nil, fmt.Errorf("Invalid task status %s", r.String)
	}

	return r.String, nil
}

type Task struct {
	ID          string `gorm:"primaryKey"`
	ProjectID   string `gorm:"index:idx_task_project"`
	Title       string
	Description *string
	Status      TaskStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time

	Assignees []Assignee `gorm:"constraint:OnDelete:CASCADE"`
	Comments  []Comment  `gorm:"constraint:OnDelete:CASCADE"`
}

package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type JSON json.RawMessage

// Scan scan value into Jsonb, implements sql.Scanner interface
func (j *JSON) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", value))
	}

	result := json.RawMessage{}
	err := json.Unmarshal(bytes, &result)
	*j = JSON(result)
	return err
}

// Value return json value, implement driver.Valuer interface
func (j JSON) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return json.RawMessage(j).MarshalJSON()
}

func (JSON) GormDBDataType(db *gorm.DB, field *schema.Field) string {

	// returns different database type based on driver name
	switch db.Dialector.Name() {
	case "mysql", "sqlite":
		return "JSON"
	case "postgres":
		return "JSONB"
	}
	return ""
}

type Notification struct {
	ID        string `gorm:"primaryKey"`
	UserID    string
	Type      string
	Body      JSON
	Read      bool
	CreatedAt time.Time
}

/*
How the Body JSON value and Type looks?

Example 1: task added
	Type: task_added
	Body: {
		"project": {
			"id": "...",
			"name": "..."
		},
		"task": {
			"id": "...",
			"title": "..."
		}
	}

Example 2: task updated
	Type: task_updated
	Body: {
		"project": {
			"id": "...",
			"name": "..."
		},
		"task": {
			"id": "...",
			"title": "..."
		},
		"updates": [
			{
				"to": "...",
				"field": "Title" / "Description" / "Status"
			},
			{
				"to": "...",
				"field": "Title" / "Description" / "Status"
			},
			...
		],
		"updater": {
			"user_id": "...",
			"username": "...",
			"email": "...",
			...
		},
	}

Example 3: assignee added
	Type: assignee_added
	Body: {
		"project": {
			"id": "...",
			"name": "..."
		},
		"task": {
			"id": "...",
			"title": "..."
		},
		"assignee": {
			"user_id": "...",
			"username": "...",
			"email": "...",
			...
		}
	}

Example 4: assignee removed
	Type: assignee_removed
	Body: {
		"project": {
			"id": "...",
			"name": "..."
		},
		"task": {
			"id": "...",
			"title": "..."
		},
		"assignee": {
			"user_id": "...",
			"username": "...",
			"email": "...",
			...
		}
	}

Example 5: join requested
	Type: join_requested
	Body: {
		"project": {
			"id": "...",
			"name": "..."
		},
		"requestor": {
			"user_id": "...",
			"username": "...",
			"email": "...",
			...
		}
	}

Example 4: join accepted/rejected
	Type: join_responded
	Body: {
		"project": {
			"id": "...",
			"name": "..."
		},
		"responder": {
			"user_id": "...",
			"username": "...",
			"email": "...",
			...
		},
		"status": "Accepted" / "Rejected"
	}

Example 5: comment added
	Type: comment_added
	Body: {
		"project": {
			"id": "...",
			"name": "..."
		},
		"task": {
			"id": "...",
			"title": "..."
		},
		"commenter": {
			"user_id": "...",
			"username": "...",
			"email": "...",
			...
		}
	}
*/

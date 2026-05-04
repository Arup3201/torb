package app

import (
	"fmt"

	"github.com/Arup3201/torb/models"
	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) error {
	var err error

	err = db.AutoMigrate(
		&models.User{},
		&models.ManualAccount{},
		&models.OauthAccount{},
		&models.Project{},
		&models.Task{},
		&models.Assignee{},
		&models.JoinRequest{},
		&models.Member{},
		&models.Comment{},
		&models.Notification{},
	)
	if err != nil {
		return fmt.Errorf("gorm db auto migrate: %w", err)
	}

	query := db.
		Table("projects p").
		Select(`p.id, 
				COUNT(t.id) FILTER (WHERE t.status='Unassigned') as unassigned_tasks,
				COUNT(t.id) FILTER (WHERE t.status='Ongoing') as ongoing_tasks, 
				COUNT(t.id) FILTER (WHERE t.status='Completed') as completed_tasks, 
				COUNT(t.id) FILTER (WHERE t.status='Abandoned') as abandoned_tasks`).
		Joins("LEFT JOIN tasks as t ON p.id=t.project_id").
		Group("p.id")
	err = db.Migrator().CreateView("project_summary", gorm.ViewOption{Query: query, Replace: true})
	if err != nil {
		return fmt.Errorf("gorm db migrator create view: %w", err)
	}

	return nil
}

package projects

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Arup3201/torb/core"
	"github.com/Arup3201/torb/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProjectSummaryRow struct {
	ID              string    `gorm:"column:id"`
	Name            string    `gorm:"column:name"`
	Description     *string   `gorm:"column:description"`
	Skills          *string   `gorm:"column:skills"`
	OwnerID         string    `gorm:"column:owner_id"`
	UnassignedTasks int64     `gorm:"column:unassigned_tasks"`
	OngoingTasks    int64     `gorm:"column:ongoing_tasks"`
	CompletedTasks  int64     `gorm:"column:completed_tasks"`
	AbandonedTasks  int64     `gorm:"column:abandoned_tasks"`
	CreatedAt       time.Time `gorm:"column:created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at"`
}

type ProjectPreviewRow struct {
	ID          string    `gorm:"column:id"`
	Name        string    `gorm:"column:name"`
	Description *string   `gorm:"column:description"`
	Skills      *string   `gorm:"column:skills"`
	OwnerID     string    `gorm:"column:owner_id"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

type ProjectRepository struct {
	db *gorm.DB
}

func NewProjectRepository(db *gorm.DB) *ProjectRepository {
	return &ProjectRepository{
		db: db,
	}
}

func (r *ProjectRepository) WithTx(tx *gorm.DB) *ProjectRepository {
	return NewProjectRepository(tx)
}

func (r *ProjectRepository) Create(ctx context.Context,
	name string,
	description, skills *string,
	ownerID string) (string, error) {

	id := uuid.NewString()
	project := models.Project{
		ID:          id,
		Name:        name,
		Description: description,
		Skills:      skills,
		OwnerID:     ownerID,
	}

	err := gorm.G[models.Project](r.db).Create(ctx, &project)
	if err != nil {
		return "", fmt.Errorf("gorm create: %w", err)
	}

	return project.ID, nil
}

func (r *ProjectRepository) Update(ctx context.Context,
	projectID string,
	name, description, skills *string) error {

	if name == nil && description == nil && skills == nil {
		return core.ErrInvalidValue
	}

	var project models.Project
	var err error
	project, err = gorm.G[models.Project](r.db).Where("id = ?", projectID).First(ctx)
	if err != nil {
		return fmt.Errorf("gorm query project: %w", err)
	}

	if name != nil {
		if strings.Trim(*name, " ") == "" {
			return core.ErrInvalidValue
		}
		project.Name = *name
	}
	if description != nil {
		project.Description = description
	}
	if skills != nil {
		project.Skills = skills
	}

	err = r.db.WithContext(ctx).Save(&project).Error
	if err != nil {
		return fmt.Errorf("gorm db save: %w", err)
	}

	return nil
}

func (r *ProjectRepository) Get(ctx context.Context, id string) (ProjectSummaryRow, error) {

	var err error
	var row ProjectSummaryRow
	err = r.db.WithContext(ctx).
		Table("projects p").
		Select(`p.id, p.name, p.description, p.skills, p.owner_id, 
				ps.unassigned_tasks, ps.ongoing_tasks, 
				ps.completed_tasks, ps.abandoned_tasks,
				p.created_at, p.updated_at
			`).
		Joins("LEFT JOIN project_summary ps ON ps.id=p.id").
		Where("p.id = ?", id).
		First(&row).
		Error
	if err == gorm.ErrRecordNotFound {
		return row, core.ErrNotFound
	} else if err != nil {
		return row, fmt.Errorf("gorm query: %w", err)
	}

	return row, nil
}

func (r *ProjectRepository) List(ctx context.Context,
	userID string) ([]ProjectSummaryRow, error) {

	var rows = []ProjectSummaryRow{}
	err := r.db.WithContext(ctx).
		Table("projects p").
		Select(`p.id, p.name, p.description, p.skills, p.owner_id, 
				ps.unassigned_tasks, ps.ongoing_tasks, ps.completed_tasks, ps.abandoned_tasks, 
				p.created_at, p.updated_at`).
		Joins("INNER JOIN members as m ON m.project_id=p.id").
		Joins("LEFT JOIN project_summary as ps ON ps.id=p.id").
		Where("m.user_id = ?", userID).
		Scan(&rows).
		Error
	if err != nil {
		return nil, fmt.Errorf("gorm db scan: %w", err)
	}

	return rows, nil
}

func (r *ProjectRepository) Public(ctx context.Context,
	userID string) ([]ProjectPreviewRow, error) {

	var rows = []ProjectPreviewRow{}
	err := r.db.WithContext(ctx).
		Table("projects").
		Select(`id, name, description, skills, owner_id, 
			created_at, updated_at`).
		Where("owner_id != ? AND NOT EXISTS (?)",
			userID,
			r.db.WithContext(ctx).
				Table("members").
				Select("1").
				Where("project_id = projects.id AND user_id = ?", userID),
		).
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("gorm db scan: %w", err)
	}

	return rows, nil
}

func (r *ProjectRepository) RecentlyCreated(ctx context.Context,
	userID string, n int) ([]ProjectSummaryRow, error) {

	var rows = []ProjectSummaryRow{}
	err := r.db.WithContext(ctx).
		Table("projects p").
		Select(`p.id, p.name, p.description, p.skills, p.owner_id, 
				ps.unassigned_tasks, ps.ongoing_tasks, ps.completed_tasks, ps.abandoned_tasks, 
				p.created_at, p.updated_at`).
		Joins("LEFT JOIN project_summary as ps ON ps.id=p.id").
		Where("p.owner_id = ?", userID).
		Order("p.created_at DESC").
		Limit(n).
		Scan(&rows).
		Error
	if err != nil {
		return nil, fmt.Errorf("gorm db scan: %w", err)
	}

	return rows, nil
}

func (r *ProjectRepository) RecentlyJoined(ctx context.Context,
	userID string,
	n int) ([]ProjectSummaryRow, error) {

	var rows = []ProjectSummaryRow{}
	err := r.db.WithContext(ctx).
		Table("projects p").
		Select(`p.id, p.name, p.description, p.skills, p.owner_id,
				ps.unassigned_tasks, ps.ongoing_tasks, ps.completed_tasks, ps.abandoned_tasks,
				p.created_at, p.updated_at`).
		Joins("INNER JOIN members as m ON m.project_id=p.id").
		Joins("LEFT JOIN project_summary as ps ON ps.id=p.id").
		Where("m.user_id = ? AND m.role = ?", userID, core.ROLE_MEMBER).
		Order("m.created_at DESC").
		Limit(n).
		Scan(&rows).
		Error
	if err != nil {
		return nil, fmt.Errorf("gorm db scan: %w", err)
	}

	return rows, nil
}

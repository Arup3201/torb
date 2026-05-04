package fixtures

import (
	"fmt"

	"github.com/Arup3201/torb/models"
	"github.com/google/uuid"
)

func RandomTaskRow(projectId, status string) models.Task {
	tId := uuid.NewString()

	desc := "Description " + tId

	return models.Task{
		ID:          tId,
		ProjectID:   projectId,
		Title:       "Test Task " + tId,
		Description: &desc,
		Status:      models.TaskStatus{String: status},
	}
}

func (f *Fixtures) InsertTask(t models.Task) string {
	if f.db != nil {
		if err := f.db.WithContext(f.ctx).Create(&t).Error; err != nil {
			panic(fmt.Sprintf("insert task fixture failed: %v", err))
		}
		return t.ID
	}
	return ""
}

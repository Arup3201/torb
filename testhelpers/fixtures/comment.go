package fixtures

import (
	"github.com/Arup3201/torb/models"
	"github.com/google/uuid"
)

func GetCommentRow(projectId, taskId, userId, content string) models.Comment {
	return models.Comment{
		ID:        uuid.NewString(),
		ProjectID: projectId,
		TaskID:    taskId,
		UserID:    userId,
		Content:   content,
	}
}

func (f *Fixtures) InsertComment(c models.Comment) {
	if f.db != nil {
		if err := f.db.WithContext(f.ctx).Create(&c).Error; err != nil {
			panic("insert comment fixture failed: " + err.Error())
		}
		return
	}
}

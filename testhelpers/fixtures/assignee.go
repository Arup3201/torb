package fixtures

import (
	"fmt"

	"github.com/Arup3201/torb/models"
)

func GetAssigneeRow(projectID, taskID, userID string) models.Assignee {
	return models.Assignee{
		ProjectID: projectID,
		TaskID:    taskID,
		UserID:    userID,
	}
}

func (f *Fixtures) InsertAssignee(a models.Assignee) {
	if f.db != nil {
		if err := f.db.WithContext(f.ctx).Create(&a).Error; err != nil {
			panic(fmt.Sprintf("insert assignee fixture failed: %v", err))
		}
		return
	}
}

func (f *Fixtures) RemoveAssignee(projectID, taskID, userID string) {
	if f.db != nil {
		if err := f.db.WithContext(f.ctx).Delete(&models.Assignee{}, "project_id = ? AND task_id = ? AND user_id = ?", projectID, taskID, userID).Error; err != nil {
			panic(fmt.Sprintf("remove assignee fixture failed: %v", err))
		}
		return
	}
}

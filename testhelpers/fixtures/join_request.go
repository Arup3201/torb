package fixtures

import (
	"fmt"

	"github.com/Arup3201/torb/core"
	"github.com/Arup3201/torb/models"
)

func GetJoinRequest(projectID, userID string) models.JoinRequest {
	return models.JoinRequest{
		ProjectID: projectID,
		UserID:    userID,
		Status:    models.JoinStatus{String: core.JOIN_STATUS_PENDING},
	}
}

func (f *Fixtures) InsertJoinRequest(j models.JoinRequest) {
	if f.db != nil {
		if err := f.db.WithContext(f.ctx).Create(&j).Error; err != nil {
			panic(fmt.Sprintf("insert join request fixture failed: %v", err))
		}
		return
	}
}

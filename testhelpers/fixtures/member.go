package fixtures

import (
	"fmt"

	"github.com/Arup3201/torb/models"
)

func GetMemberRow(projectID, userID, role string) models.Member {
	return models.Member{
		ProjectID: projectID,
		UserID:    userID,
		Role:      models.UserRole{String: role},
	}
}

func (f *Fixtures) InsertMember(m models.Member) {
	if f.db != nil {
		if err := f.db.WithContext(f.ctx).Create(&m).Error; err != nil {
			panic(fmt.Sprintf("insert member fixture failed: %v", err))
		}
		return
	}
}

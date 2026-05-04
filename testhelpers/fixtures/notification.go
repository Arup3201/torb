package fixtures

import (
	"encoding/json"
	"fmt"

	"github.com/Arup3201/torb/models"
	"github.com/google/uuid"
)

func GetNotificationRow(userID, notificationType string, body any) models.Notification {
	id := uuid.NewString()

	jsonBody, err := json.Marshal(body)
	if err != nil {
		panic(fmt.Sprintf("json marshal body failed: %v", err))
	}

	return models.Notification{
		ID:     id,
		UserID: userID,
		Type:   notificationType,
		Body:   jsonBody,
	}
}

func (f *Fixtures) InsertNotification(n models.Notification) {
	if f.db != nil {
		if err := f.db.WithContext(f.ctx).Create(&n).Error; err != nil {
			panic(fmt.Sprintf("insert notification fixture failed: %v", err))
		}
		return
	}
}

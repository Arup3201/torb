package notifications

import (
	"context"
	"fmt"

	"github.com/Arup3201/torb/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type NotificationRepository struct {
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) *NotificationRepository {
	return &NotificationRepository{
		db: db,
	}
}

func (r *NotificationRepository) Create(ctx context.Context,
	userID, nType string,
	body models.JSON,
	read bool) (string, error) {

	id := uuid.NewString()
	notification := models.Notification{
		ID:     id,
		UserID: userID,
		Type:   nType,
		Body:   body,
		Read:   read,
	}

	err := gorm.G[models.Notification](r.db).Create(ctx, &notification)
	if err != nil {
		return "", fmt.Errorf("gorm create: %w", err)
	}

	return notification.ID, nil
}

func (r *NotificationRepository) Update(ctx context.Context,
	id string,
	read bool) error {

	var err error
	var notification models.Notification

	notification, err =
		gorm.G[models.Notification](r.db).
			Where("id = ?", id).
			First(ctx)
	if err != nil {
		return fmt.Errorf("gorm query notification: %w", err)
	}

	notification.Read = read

	err = r.db.Save(&notification).Error
	if err != nil {
		return fmt.Errorf("gorm db save: %w", err)
	}

	return nil
}

func (r *NotificationRepository) List(ctx context.Context,
	userID string) ([]models.Notification, error) {

	notifications, err :=
		gorm.G[models.Notification](r.db).
			Where("user_id = ?", userID).
			Find(ctx)
	if err != nil {
		return nil, fmt.Errorf("gorm query: %w", err)
	}

	return notifications, nil
}

package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Arup3201/torb/core"
	"github.com/Arup3201/torb/notifications"
)

type ListedMessages struct {
	Messages []notifications.Notification `json:"messages"`
}

type MessageApi struct {
	notificationService *notifications.NotificationService
}

func NewMessageApi(
	notificationService *notifications.NotificationService,
) *MessageApi {
	return &MessageApi{
		notificationService: notificationService,
	}
}

func (api *MessageApi) List(w http.ResponseWriter, r *http.Request) error {

	userID, err := GetUserID(r)
	if err != nil {
		return fmt.Errorf("get user Id: %w", err)
	}

	notifications, err := api.notificationService.List(
		r.Context(),
		userID,
	)
	if err != nil {
		return fmt.Errorf("notification service List: %w", err)
	}

	json.NewEncoder(w).Encode(HTTPSuccessResponse[ListedMessages]{
		Data: &ListedMessages{
			Messages: notifications,
		},
	})

	return nil
}

func (api *MessageApi) MarkAsRead(w http.ResponseWriter, r *http.Request) error {

	notificationID := r.PathValue("id")
	if notificationID == "" {
		return core.ErrInvalidValue
	}

	userID, err := GetUserID(r)
	if err != nil {
		return fmt.Errorf("get user Id: %w", err)
	}

	err = api.notificationService.MarkAsRead(
		r.Context(),
		userID,
		notificationID,
	)
	if err != nil {
		return fmt.Errorf("notification service MarkAsRead: %w", err)
	}

	json.NewEncoder(w).Encode(HTTPSuccessResponse[ListedMessages]{
		Message: "Notification marked as read",
	})

	return nil
}

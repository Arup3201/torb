package users

import (
	"context"
	"fmt"
	"time"
)

type User struct {
	ID            string    `json:"id"`
	Username      string    `json:"username"`
	DisplayName   *string   `json:"display_name"` // nullable
	Email         string    `json:"email"`
	AvatarURL     *string   `json:"avatar_url"` // nullable
	Skills        string    `json:"skills"`
	Timestamp     string    `json:"timestamp"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	LastLoginTime time.Time `json:"-"`
}

type ProjectTaskCount struct {
	Projects       int64 `json:"projects"`
	Tasks          int64 `json:"tasks"`
	CompletedTasks int64 `json:"completed_tasks"`
}

type UpdateUserBody struct {
	Username    *string
	DisplayName *string
	AvatarURL   *string
	Skills      *string
	Timestamp   *string
}

type UserService struct {
	userRepo *UserRepository
}

func NewUserService(userRepo *UserRepository) *UserService {
	return &UserService{
		userRepo: userRepo,
	}
}

func (s *UserService) Get(ctx context.Context,
	id string) (*User, error) {

	user, err := s.userRepo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("user repository get: %w", err)
	}

	return &User{
		ID:            user.ID,
		Username:      user.Username,
		Email:         user.Email,
		DisplayName:   user.DisplayName,
		AvatarURL:     user.AvatarURL,
		Skills:        user.Skills,
		Timestamp:     user.Timestamp,
		CreatedAt:     user.CreatedAt,
		UpdatedAt:     user.UpdatedAt,
		LastLoginTime: user.LastLoginTime,
	}, nil
}

func (s *UserService) ProjectAndTaskCount(ctx context.Context,
	id string) (*ProjectTaskCount, error) {

	projectsCnt, err := s.userRepo.CountProjects(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("user repository CountProjects: %w", err)
	}

	tasksCnt, err := s.userRepo.CountTasks(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("user repository CountTasks: %w", err)
	}

	completedTasksCnt, err := s.userRepo.CountCompletedTasks(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("user repository CountCompletedTasks: %w", err)
	}

	return &ProjectTaskCount{
		Projects:       projectsCnt,
		Tasks:          tasksCnt,
		CompletedTasks: completedTasksCnt,
	}, nil
}

func (s *UserService) Update(ctx context.Context,
	id string, update UpdateUserBody) error {

	user, err := s.userRepo.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("user repository Get: %w", err)
	}

	if update.Username != nil {
		user.Username = *update.Username
	}
	if update.DisplayName != nil {
		user.DisplayName = update.DisplayName
	}
	if update.AvatarURL != nil {
		user.AvatarURL = update.AvatarURL
	}
	if update.Skills != nil {
		user.Skills = *update.Skills
	}
	if update.Timestamp != nil {
		user.Timestamp = *update.Timestamp
	}

	if update.Username != nil ||
		update.DisplayName != nil ||
		update.AvatarURL != nil ||
		update.Skills != nil ||
		update.Timestamp != nil {
		err = s.userRepo.Update(ctx, id, user)
		if err != nil {
			return fmt.Errorf("user repository Update: %w", err)
		}
	}

	return nil
}

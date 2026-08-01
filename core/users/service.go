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
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	LastLoginTime time.Time `json:"-"`
}

type ProjectTaskCount struct {
	Projects int64 `json:"projects"`
	Tasks    int64 `json:"tasks"`
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

	completedTasksCnt, err := s.userRepo.CountCompletedTasks(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("user repository CountCompletedTasks: %w", err)
	}

	return &ProjectTaskCount{
		Projects: projectsCnt,
		Tasks:    completedTasksCnt,
	}, nil
}

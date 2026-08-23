package users

import (
	"context"
	"fmt"
	"mime/multipart"
	"time"

	"github.com/Arup3201/torb/core/documents"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

type User struct {
	ID            string    `json:"id"`
	Username      string    `json:"username"`
	DisplayName   *string   `json:"display_name"` // nullable
	Email         string    `json:"email"`
	AvatarKey     *string   `json:"avatar_url"` // nullable
	Skills        string    `json:"skills"`
	Timezone      string    `json:"timezone"`
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
	AvatarKey   *string
	Skills      *string
	Timezone    *string
}

type UserService struct {
	userRepo     *UserRepository
	documentRepo *documents.DocumentRepository
	s3Client     *s3.Client
}

func NewUserService(
	userRepo *UserRepository,
	docRepo *documents.DocumentRepository,
	s3Client *s3.Client) *UserService {
	return &UserService{
		userRepo:     userRepo,
		documentRepo: docRepo,
		s3Client:     s3Client,
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
		AvatarKey:     user.AvatarKey,
		Skills:        user.Skills,
		Timezone:      user.Timezone,
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
	if update.AvatarKey != nil {
		user.AvatarKey = update.AvatarKey
	}
	if update.Skills != nil {
		user.Skills = *update.Skills
	}
	if update.Timezone != nil {
		user.Timezone = *update.Timezone
	}

	if update.Username != nil ||
		update.DisplayName != nil ||
		update.AvatarKey != nil ||
		update.Skills != nil ||
		update.Timezone != nil {
		err = s.userRepo.Update(ctx, id, user)
		if err != nil {
			return fmt.Errorf("user repository Update: %w", err)
		}
	}

	return nil
}

func (s *UserService) UploadAvatar(ctx context.Context,
	userID string,
	avatar multipart.File,
	filename, fileContentType string,
	size uint, // Bytes
) error {

	avatarFileKey := fmt.Sprintf("users/%s/avatars/%s", userID, filename)
	_, err := s.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String("torb"),
		Key:         aws.String(avatarFileKey),
		Body:        avatar,
		ContentType: aws.String(fileContentType),
	})
	if err != nil {
		return err
	}

	id := uuid.NewString()
	err = s.documentRepo.Create(ctx, id, avatarFileKey, "avatar", fileContentType, size)
	if err != nil {
		return err
	}

	err = s.Update(ctx, userID, UpdateUserBody{
		AvatarKey: &avatarFileKey,
	})
	if err != nil {
		return err
	}

	return nil
}

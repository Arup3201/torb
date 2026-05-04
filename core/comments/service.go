package comments

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Arup3201/torb/core"
	"github.com/Arup3201/torb/core/members"
)

type Comment struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	TaskID      string    `json:"task_id"`
	Content     string    `json:"content"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	core.Avatar `json:"avatar"`
}

type CommentService struct {
	commentRepo *CommentRepository
	memberRepo  *members.MemberRepository
}

func NewCommentService(commentRepo *CommentRepository,
	memberRepo *members.MemberRepository) *CommentService {
	return &CommentService{
		commentRepo: commentRepo,
		memberRepo:  memberRepo,
	}
}

func (s *CommentService) Create(ctx context.Context,
	projectID, taskID, userID string,
	comment string) (string, error) {

	var err error

	err = core.NeedsToBeAMember(ctx, s.memberRepo, projectID, userID)
	if err != nil {
		return "", fmt.Errorf("needs to be a member: %w", err)
	}

	if strings.Trim(comment, " ") == "" {
		return "", core.ErrInvalidValue
	}

	commentID, err := s.commentRepo.Create(ctx, projectID, taskID, userID, comment)
	if err != nil {
		return "", fmt.Errorf("comment repository create: %w", err)
	}

	return commentID, nil
}

func (s *CommentService) List(ctx context.Context,
	projectID, taskID, userID string) ([]Comment, error) {

	var err error

	err = core.NeedsToBeAMember(ctx, s.memberRepo, projectID, userID)
	if err != nil {
		return nil, fmt.Errorf("needs to be a member: %w", err)
	}

	rows, err := s.commentRepo.List(ctx, projectID, taskID)
	if err != nil {
		return nil, fmt.Errorf("comment repository comments: %w", err)
	}

	comments := []Comment{}
	for _, r := range rows {
		comments = append(comments, Comment{
			ID:        r.ID,
			ProjectID: r.ProjectID,
			TaskID:    r.TaskID,
			Content:   r.Content,
			CreatedAt: r.CreatedAt,
			UpdatedAt: r.UpdatedAt,
			Avatar: core.Avatar{
				UserID:      r.UserID,
				Username:    r.Username,
				DisplayName: r.DisplayName,
				Email:       r.Email,
				AvatarURL:   r.AvatarURL,
			},
		})
	}

	return comments, nil
}

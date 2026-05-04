package members

import (
	"context"
	"fmt"
	"time"

	"github.com/Arup3201/torb/core"
)

type Member struct {
	ProjectID   string    `json:"project_id"`
	Role        string    `json:"role"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	core.Avatar `json:"avatar"`
}

type MemberService struct {
	memberRepo *MemberRepository
}

func NewMemberService(memberRepo *MemberRepository) *MemberService {
	return &MemberService{
		memberRepo: memberRepo,
	}
}

func (s *MemberService) GetRole(ctx context.Context,
	projectID, userID string) (string, error) {

	role, err := s.memberRepo.Role(ctx, projectID, userID)
	if err != nil {
		return "", fmt.Errorf("member repository role: %w", err)
	}

	return role, nil
}

func (s *MemberService) Count(ctx context.Context,
	projectID, userID string) (int64, error) {

	count, err := s.memberRepo.Count(ctx, projectID)
	if err != nil {
		return -1, fmt.Errorf("member repository count: %w", err)
	}

	return count, nil
}

func (s *MemberService) AllMembers(ctx context.Context,
	projectID, userID string) ([]Member, error) {

	var err error

	err = core.NeedsToBeAMember(ctx, s.memberRepo, projectID, userID)
	if err != nil {
		return nil, fmt.Errorf("needs to be a member: %w", err)
	}

	rows, err := s.memberRepo.List(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("member repository list: %w", err)
	}

	members := []Member{}
	for _, r := range rows {
		members = append(members, Member{
			ProjectID: r.ProjectID,
			Role:      r.Role.String,
			CreatedAt: r.CreatedAt,
			UpdatedAt: r.UpdatedAt,
			Avatar: core.Avatar{
				UserID:      r.UserID,
				Username:    r.Username,
				Email:       r.Email,
				DisplayName: r.DisplayName,
				AvatarURL:   r.AvatarURL,
			},
		})
	}

	return members, nil
}

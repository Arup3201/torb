package requests

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/Arup3201/torb/core"
	"github.com/Arup3201/torb/core/members"
	"gorm.io/gorm"
)

type JoinRequest struct {
	ProjectID   string    `json:"project_id"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	core.Avatar `json:"avatar"`
}

type JoinRequestService struct {
	txManager  *core.TxManager
	joinRepo   *JoinRepository
	memberRepo *members.MemberRepository
}

func NewJoinRequestService(txManager *core.TxManager,
	joinRepo *JoinRepository,
	memberRepo *members.MemberRepository) *JoinRequestService {
	return &JoinRequestService{
		txManager:  txManager,
		joinRepo:   joinRepo,
		memberRepo: memberRepo,
	}
}

func (s *JoinRequestService) Create(ctx context.Context,
	projectID, userID string) error {

	err := s.joinRepo.Create(ctx, projectID, userID, core.JOIN_STATUS_PENDING)
	if err != nil {
		return fmt.Errorf("join repository create: %w", err)
	}

	return nil
}

func (s *JoinRequestService) GetStatus(ctx context.Context,
	projectID, userID string) (string, error) {

	status, err := s.joinRepo.Status(ctx, projectID, userID)
	if errors.Is(err, core.ErrNotFound) {
		status = "Not Requested"
	} else if err != nil {
		return "", fmt.Errorf("join repository status: %w", err)
	}

	return status, nil
}

func (s *JoinRequestService) List(ctx context.Context,
	projectID, userID string) ([]JoinRequest, error) {

	var err error

	err = core.NeedsToBeAnOwner(ctx, s.memberRepo, projectID, userID)
	if err != nil {
		return nil, fmt.Errorf("needs to be an owner: %w", err)
	}

	rows, err := s.joinRepo.List(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("join request repository list: %w", err)
	}

	requests := []JoinRequest{}
	for _, r := range rows {
		requests = append(requests, JoinRequest{
			ProjectID: r.ProjectID,
			Status:    r.Status.String,
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

	return requests, nil
}

func (s *JoinRequestService) Respond(ctx context.Context,
	projectID, responderID, requestorID, joinStatus string) error {

	var err error

	err = core.NeedsToBeAnOwner(ctx, s.memberRepo, projectID, responderID)
	if err != nil {
		return fmt.Errorf("needs to be an owner: %w", err)
	}

	if !slices.Contains([]string{
		core.JOIN_STATUS_PENDING,
		core.JOIN_STATUS_ACCEPTED,
		core.JOIN_STATUS_REJECTED,
	}, joinStatus) {
		return core.ErrInvalidValue
	}

	status, err := s.joinRepo.Status(ctx, projectID, requestorID)
	if err != nil {
		return fmt.Errorf("join request repository status: %w", err)
	}

	if status == joinStatus {
		return core.ErrInvalidValue
	}

	// Y Rejected -> Pending
	// X Rejected -> Accepted
	// X Accepted -> Pending|Rejected

	if (status == core.JOIN_STATUS_REJECTED &&
		joinStatus == core.JOIN_STATUS_ACCEPTED) ||
		(status == core.JOIN_STATUS_ACCEPTED &&
			slices.Contains([]string{
				core.JOIN_STATUS_PENDING,
				core.JOIN_STATUS_REJECTED,
			}, joinStatus)) {
		return core.ErrInvalidValue
	}

	err = s.txManager.WithTx(func(tx *gorm.DB) error {
		joinRepo := s.joinRepo.WithTx(tx)
		memberRepo := s.memberRepo.WithTx(tx)

		err = joinRepo.Update(ctx, projectID, requestorID, joinStatus)
		if err != nil {
			return fmt.Errorf("join repository update: %w", err)
		}

		if joinStatus == core.JOIN_STATUS_ACCEPTED {
			err = memberRepo.Create(ctx, projectID, requestorID, core.ROLE_MEMBER)
			if err != nil {
				return fmt.Errorf("member repository create: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("txManager WithTx: %w", err)
	}

	return nil
}

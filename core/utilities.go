package core

import (
	"context"
	"fmt"
)

type RoleChecker interface {

	/*
		Check user role in a project projectID for user userID
	*/
	Role(ctx context.Context,
		projectID string, userID string) (string, error)
}

func NeedsToBeAMember(ctx context.Context,
	c RoleChecker,
	projectID, userID string) error {

	_, err := c.Role(ctx, projectID, userID)
	if err == ErrNotFound {
		return ErrForbidden
	} else if err != nil {
		return fmt.Errorf("role checker Role: %w", err)
	}

	return nil
}

func NeedsToBeAnOwner(ctx context.Context,
	c RoleChecker,
	projectID, userID string) error {

	role, err := c.Role(ctx, projectID, userID)
	if err == ErrNotFound {
		return ErrForbidden
	} else if err != nil {
		return fmt.Errorf("role checker Role: %w", err)
	}

	if role != ROLE_OWNER {
		return ErrForbidden
	}

	return nil
}

type AssigneeValidator interface {

	/*
		Check if user userID is an assignee of the task taskID
	*/
	Is(ctx context.Context,
		projectID, taskID, userID string) error
}

func NeedsToBeAnAssignee(ctx context.Context,
	v AssigneeValidator,
	projectID, taskID, userID string) error {

	var err error
	err = v.Is(ctx, projectID, taskID, userID)
	if err == ErrNotFound {
		return ErrForbidden
	} else if err != nil {
		return fmt.Errorf("assignee validator Is: %w", err)
	}

	return nil
}

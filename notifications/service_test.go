package notifications

import (
	"context"
	"encoding/json"
	"log"
	"testing"

	"github.com/Arup3201/torb/core"
	"github.com/Arup3201/torb/core/members"
	"github.com/Arup3201/torb/core/projects"
	"github.com/Arup3201/torb/core/tasks"
	"github.com/Arup3201/torb/core/users"
	"github.com/Arup3201/torb/models"
	"github.com/Arup3201/torb/testdata"
	"github.com/Arup3201/torb/testhelpers"
	"github.com/Arup3201/torb/testhelpers/fixtures"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var USER_ONE, USER_TWO, USER_THREE string

type notificationServiceTestSuite struct {
	suite.Suite
	ctx         context.Context
	pgContainer *testhelpers.PostgresContainer
	db          *gorm.DB
	fixtures    *fixtures.Fixtures
	service     *NotificationService
}

func TestNotificationService(t *testing.T) {
	suite.Run(t, new(notificationServiceTestSuite))
}

func (suite *notificationServiceTestSuite) SetupSuite() {
	var err error

	suite.ctx = context.Background()

	suite.pgContainer, err = testhelpers.CreatePostgresContainer(suite.ctx)
	if err != nil {
		log.Fatal(err)
	}

	suite.db, err = gorm.Open(postgres.Open(suite.pgContainer.ConnectionString), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	err = testdata.TestMigrate(suite.db)
	if err != nil {
		log.Fatal(err)
	}

	suite.db.AutoMigrate(&models.Notification{})

	projectRepo := projects.NewProjectRepository(suite.db)
	taskRepo := tasks.NewTaskRepository(suite.db)
	memberRepo := members.NewMemberRepository(suite.db)
	userRepo := users.NewUserRepository(suite.db)
	notificationRepo := NewNotificationRepository(suite.db)
	suite.service = NewNotificationService(
		projectRepo,
		taskRepo,
		memberRepo,
		userRepo,
		notificationRepo,
	)

	suite.fixtures = fixtures.New(suite.ctx, suite.db)

	USER_ONE = suite.fixtures.InsertUser(fixtures.RandomUserRow())
	USER_TWO = suite.fixtures.InsertUser(fixtures.RandomUserRow())
	USER_THREE = suite.fixtures.InsertUser(fixtures.RandomUserRow())
}

func (suite *notificationServiceTestSuite) Cleanup() {
	var err error
	err = suite.db.WithContext(suite.ctx).
		Exec("DELETE FROM projects").Error
	suite.Require().NoError(err)

	err = suite.db.WithContext(suite.ctx).
		Exec("DELETE FROM notifications").Error
	suite.Require().NoError(err)
}

func (suite *notificationServiceTestSuite) TestTaskAdded() {
	t := suite.T()

	t.Run("should create notification for task_added", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertMember(fixtures.GetMemberRow(p, USER_TWO, core.ROLE_MEMBER))
		taskID := suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_UNASSIGNED))
		_, _, err := suite.service.TaskAdded(suite.ctx, p, taskID)

		suite.Cleanup()
		suite.Require().NoError(err)
	})
	t.Run("should create notifications for all members", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertMember(fixtures.GetMemberRow(p, USER_TWO, core.ROLE_MEMBER))
		taskID := suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_UNASSIGNED))
		suite.service.TaskAdded(suite.ctx, p, taskID)

		n, _ :=
			gorm.G[models.Notification](suite.db).
				Where("type = ?", NT_TASK_ADDED).
				Find(suite.ctx)

		suite.Cleanup()
		suite.Require().Equal(1, len(n))
	})
	t.Run("should match task_added notification body", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertMember(fixtures.GetMemberRow(p, USER_TWO, core.ROLE_MEMBER))
		taskID := suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_UNASSIGNED))
		suite.service.TaskAdded(suite.ctx, p, taskID)

		/*
			Body: {
					"project": {
						"id": "...",
						"name": "..."
					},
					"task": {
						"id": "...",
						"title": "..."
					}
				}
		*/
		n, _ :=
			gorm.G[models.Notification](suite.db).
				Where("user_id = ?", USER_TWO).
				First(suite.ctx)
		suite.Cleanup()
		var body TaskAdded
		err := json.Unmarshal(n.Body, &body)
		suite.Require().NoError(err)
		suite.Require().Equal(p, body.Project.ID)
		suite.Require().Equal(taskID, body.Task.ID)
	})
}

func (suite *notificationServiceTestSuite) TestTaskUpdated() {
	t := suite.T()

	t.Run("should create notification for task_updated", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertMember(fixtures.GetMemberRow(p, USER_TWO, core.ROLE_MEMBER))
		taskID := suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_UNASSIGNED))
		suite.fixtures.InsertAssignee(fixtures.GetAssigneeRow(p, taskID, USER_TWO))

		_, _, err := suite.service.TaskUpdated(suite.ctx, p, taskID, &[]string{"New title"}[0], nil, nil, USER_TWO)

		suite.Cleanup()
		suite.Require().NoError(err)
	})
	t.Run("should create notification for owner", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertMember(fixtures.GetMemberRow(p, USER_TWO, core.ROLE_MEMBER))
		taskID := suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_UNASSIGNED))
		suite.fixtures.InsertAssignee(fixtures.GetAssigneeRow(p, taskID, USER_TWO))

		suite.service.TaskUpdated(suite.ctx, p, taskID, &[]string{"New title"}[0], nil, nil, USER_TWO)

		n, _ :=
			gorm.G[models.Notification](suite.db).
				Where("type = ?", NT_TASK_UPDATED).
				Find(suite.ctx)
		suite.Cleanup()
		suite.Require().Equal(1, len(n))
		suite.Require().Equal(USER_ONE, n[0].UserID)
	})
	t.Run("should match notification body", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertMember(fixtures.GetMemberRow(p, USER_TWO, core.ROLE_MEMBER))
		taskID := suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_UNASSIGNED))
		suite.fixtures.InsertAssignee(fixtures.GetAssigneeRow(p, taskID, USER_TWO))
		suite.service.TaskUpdated(suite.ctx, p, taskID, &[]string{"New title"}[0], nil, nil, USER_TWO)

		n, _ :=
			gorm.G[models.Notification](suite.db).
				Where("user_id = ?", USER_ONE).
				First(suite.ctx)

		suite.Cleanup()
		var body TaskUpdated
		err := json.Unmarshal(n.Body, &body)
		suite.Require().NoError(err)
		suite.Require().Equal(p, body.Project.ID)
		suite.Require().Equal(taskID, body.Task.ID)
		suite.Require().Equal(1, len(body.Updates))
		suite.Require().Equal("New title", body.Updates[0].To)
		suite.Require().Equal("Title", body.Updates[0].Field)
		suite.Require().Equal(USER_TWO, body.Updater.UserID)
	})
	t.Run("should have multiple updates", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertMember(fixtures.GetMemberRow(p, USER_TWO, core.ROLE_MEMBER))
		taskID := suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_UNASSIGNED))
		suite.fixtures.InsertAssignee(fixtures.GetAssigneeRow(p, taskID, USER_TWO))
		suite.service.TaskUpdated(suite.ctx, p, taskID, &[]string{"New title"}[0], &[]string{"New description"}[0], nil, USER_TWO)

		n, _ :=
			gorm.G[models.Notification](suite.db).
				Where("user_id = ?", USER_ONE).
				First(suite.ctx)

		suite.Cleanup()
		var body TaskUpdated
		err := json.Unmarshal(n.Body, &body)
		suite.Require().NoError(err)
		suite.Require().Equal(p, body.Project.ID)
		suite.Require().Equal(taskID, body.Task.ID)
		suite.Require().Equal(2, len(body.Updates))
	})
}

func (suite *notificationServiceTestSuite) TestAssigneeUpdated() {
	t := suite.T()

	t.Run("should create assignee_added notification", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertMember(fixtures.GetMemberRow(p, USER_TWO, core.ROLE_MEMBER))
		taskID := suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_UNASSIGNED))
		suite.fixtures.InsertAssignee(fixtures.GetAssigneeRow(p, taskID, USER_TWO))

		_, _, err := suite.service.AssigneeUpdated(suite.ctx, p, taskID, USER_TWO, true)

		suite.Cleanup()
		suite.Require().NoError(err)
	})
	t.Run("should be assignee_added type of notification", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertMember(fixtures.GetMemberRow(p, USER_TWO, core.ROLE_MEMBER))
		taskID := suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_UNASSIGNED))
		suite.fixtures.InsertAssignee(fixtures.GetAssigneeRow(p, taskID, USER_TWO))
		suite.service.AssigneeUpdated(suite.ctx, p, taskID, USER_TWO, true)

		n, _ :=
			gorm.G[models.Notification](suite.db).
				Where("user_id = ?", USER_TWO).
				First(suite.ctx)

		suite.Cleanup()
		suite.Require().Equal(NT_ASSIGNEE_ADDED, n.Type)
	})
	t.Run("should be assignee_removed type of notification", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertMember(fixtures.GetMemberRow(p, USER_TWO, core.ROLE_MEMBER))
		suite.fixtures.InsertMember(fixtures.GetMemberRow(p, USER_THREE, core.ROLE_MEMBER))
		taskID := suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_UNASSIGNED))
		suite.fixtures.InsertAssignee(fixtures.GetAssigneeRow(p, taskID, USER_TWO))
		suite.fixtures.InsertAssignee(fixtures.GetAssigneeRow(p, taskID, USER_THREE))
		suite.fixtures.RemoveAssignee(p, taskID, USER_TWO)
		suite.service.AssigneeUpdated(suite.ctx, p, taskID, USER_TWO, false)

		n, _ :=
			gorm.G[models.Notification](suite.db).
				Where("user_id = ?", USER_THREE).
				First(suite.ctx)

		suite.Cleanup()
		suite.Require().Equal(NT_ASSIGNEE_REMOVED, n.Type)
	})
	t.Run("should match notification body", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertMember(fixtures.GetMemberRow(p, USER_TWO, core.ROLE_MEMBER))
		taskID := suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_UNASSIGNED))
		suite.fixtures.InsertAssignee(fixtures.GetAssigneeRow(p, taskID, USER_TWO))
		suite.service.AssigneeUpdated(suite.ctx, p, taskID, USER_TWO, true)

		n, _ :=
			gorm.G[models.Notification](suite.db).
				Where("user_id = ?", USER_TWO).
				First(suite.ctx)

		suite.Cleanup()
		var body AssigneeUpdated
		err := json.Unmarshal(n.Body, &body)
		suite.Require().NoError(err)
		suite.Require().Equal(p, body.Project.ID)
		suite.Require().Equal(taskID, body.Task.ID)
		suite.Require().Equal(USER_TWO, body.Assignee.UserID)
	})
}

func (suite *notificationServiceTestSuite) TestJoinRequested() {
	t := suite.T()

	t.Run("should create join_requested notification", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))

		_, _, err := suite.service.JoinRequested(suite.ctx, p, USER_TWO)

		n, _ :=
			gorm.G[models.Notification](suite.db).
				Where("user_id = ?", USER_ONE).
				First(suite.ctx)
		suite.Cleanup()
		suite.Require().NoError(err)
		suite.Require().Equal(NT_JOIN_REQUESTED, n.Type)

		var body JoinRequested
		err = json.Unmarshal(n.Body, &body)
		suite.Require().NoError(err)
		suite.Require().Equal(p, body.Project.ID)
		suite.Require().Equal(USER_TWO, body.Requestor.UserID)
	})
}

func (suite *notificationServiceTestSuite) TestJoinResponded() {
	t := suite.T()

	t.Run("should create join_responded notification", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertJoinRequest(models.JoinRequest{
			ProjectID: p,
			UserID:    USER_TWO,
			Status: models.JoinStatus{
				String: core.JOIN_STATUS_ACCEPTED,
			},
		})

		_, _, err := suite.service.JoinResponded(suite.ctx, p, USER_TWO, core.JOIN_STATUS_ACCEPTED)

		n, _ :=
			gorm.G[models.Notification](suite.db).
				Where("user_id = ?", USER_TWO).
				First(suite.ctx)
		suite.Cleanup()
		suite.Require().NoError(err)
		suite.Require().Equal(NT_JOIN_RESPONDED, n.Type)

		var body JoinResponded
		err = json.Unmarshal(n.Body, &body)
		suite.Require().NoError(err)
		suite.Require().Equal(p, body.Project.ID)
		suite.Require().Equal(USER_ONE, body.Responder.UserID)
		suite.Require().Equal(core.JOIN_STATUS_ACCEPTED, body.Status)
	})
}

func (suite *notificationServiceTestSuite) TestCommentAdded() {
	t := suite.T()

	t.Run("should create comment_added notification", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertMember(fixtures.GetMemberRow(p, USER_TWO, core.ROLE_MEMBER))
		taskID := suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_UNASSIGNED))
		suite.fixtures.InsertAssignee(fixtures.GetAssigneeRow(p, taskID, USER_TWO))

		_, _, err := suite.service.CommentAdded(suite.ctx, p, taskID, USER_TWO)

		n, _ :=
			gorm.G[models.Notification](suite.db).
				Where("user_id = ?", USER_TWO).
				First(suite.ctx)
		suite.Cleanup()
		suite.Require().NoError(err)
		suite.Require().Equal(NT_COMMENT_ADDED, n.Type)

		var body CommentAdded
		err = json.Unmarshal(n.Body, &body)
		suite.Require().NoError(err)
		suite.Require().Equal(p, body.Project.ID)
		suite.Require().Equal(taskID, body.Task.ID)
		suite.Require().Equal(USER_TWO, body.Commenter.UserID)
	})
}

func (suite *notificationServiceTestSuite) TestList() {
	t := suite.T()

	t.Run("should list notifications", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertNotification(fixtures.GetNotificationRow(USER_ONE, NT_JOIN_REQUESTED, JoinRequested{
			Project: ProjectBody{
				ID: p,
			},
			Requestor: core.Avatar{
				UserID: USER_TWO,
			},
		}))
		suite.fixtures.InsertNotification(fixtures.GetNotificationRow(USER_ONE, NT_JOIN_REQUESTED, JoinRequested{
			Project: ProjectBody{
				ID: p,
			},
			Requestor: core.Avatar{
				UserID: USER_THREE,
			},
		}))

		notifications, err := suite.service.List(suite.ctx, USER_ONE)

		suite.Cleanup()
		suite.Require().NoError(err)
		suite.Require().Equal(2, len(notifications))
	})
}

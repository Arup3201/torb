package tasks

import (
	"context"
	"log"
	"testing"

	"github.com/Arup3201/torb/core"
	"github.com/Arup3201/torb/models"
	"github.com/Arup3201/torb/testdata"
	"github.com/Arup3201/torb/testhelpers"
	"github.com/Arup3201/torb/testhelpers/fixtures"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var USER_ONE, USER_TWO, USER_THREE string

type taskRepositoryTestSuite struct {
	suite.Suite
	pgContainer *testhelpers.PostgresContainer
	db          *gorm.DB
	fixtures    *fixtures.Fixtures
	repo        *TaskRepository
	ctx         context.Context
}

func (suite *taskRepositoryTestSuite) SetupSuite() {
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

	suite.repo = NewTaskRepository(suite.db)
	suite.fixtures = fixtures.New(suite.ctx, suite.db)

	USER_ONE = suite.fixtures.InsertUser(fixtures.RandomUserRow())
	USER_TWO = suite.fixtures.InsertUser(fixtures.RandomUserRow())
	USER_THREE = suite.fixtures.InsertUser(fixtures.RandomUserRow())
}

func (suite *taskRepositoryTestSuite) Cleanup() {
	err := suite.db.WithContext(suite.ctx).
		Exec("DELETE FROM projects").Error
	suite.Require().NoError(err)
}

func TestTaskRepository(t *testing.T) {
	suite.Run(t, new(taskRepositoryTestSuite))
}

func (suite *taskRepositoryTestSuite) TestTaskCreate() {
	t := suite.T()

	t.Run("should create task", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		sample_title := "sample task"
		sample_description := "sample description"
		sample_status := core.TASK_STATUS_UNASSIGNED

		_, err := suite.repo.Create(suite.ctx, p,
			sample_title, sample_description, sample_status)

		suite.Cleanup()

		suite.Require().NoError(err)
	})
	t.Run("should create task with title description and status", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		sample_title := "sample task"
		sample_description := "sample description"
		sample_status := core.TASK_STATUS_UNASSIGNED

		id, _ := suite.repo.Create(suite.ctx, p,
			sample_title, sample_description, sample_status)
		task, _ := gorm.G[models.Task](suite.db).Where("id = ?", id).First(suite.ctx)

		suite.Cleanup()

		suite.Require().Equal(id, task.ID)
		suite.Require().Equal(sample_title, task.Title)
		suite.Require().Equal(sample_description, *task.Description)
		suite.Require().Equal(sample_status, task.Status.String)
	})
	t.Run("should create task with empty description", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		sample_title := "sample task"
		sample_status := core.TASK_STATUS_UNASSIGNED

		_, err := suite.repo.Create(suite.ctx, p,
			sample_title, "", sample_status)

		suite.Cleanup()

		suite.Require().NoError(err)
	})
}

func (suite *taskRepositoryTestSuite) TestTaskGet() {
	t := suite.T()

	t.Run("should get title description and status of the task", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		sample_title := "sample task"
		sample_description := "sample description"
		sample_status := core.TASK_STATUS_UNASSIGNED
		taskID := suite.fixtures.InsertTask(models.Task{
			ProjectID:   p,
			Title:       sample_title,
			Description: &sample_description,
			Status:      models.TaskStatus{String: sample_status},
		})
		task, _ := suite.repo.Get(suite.ctx, taskID)

		suite.Cleanup()

		suite.Require().Equal(taskID, task.ID)
		suite.Require().Equal(sample_title, task.Title)
		suite.Require().Equal(sample_description, *task.Description)
		suite.Require().Equal(sample_status, task.Status.String)
	})
	t.Run("should list assignees", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertMember(fixtures.GetMemberRow(p, USER_TWO, core.ROLE_MEMBER))
		taskID := suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_UNASSIGNED))
		suite.fixtures.InsertAssignee(fixtures.GetAssigneeRow(p, taskID, USER_TWO))

		_, err := suite.repo.Get(suite.ctx, taskID)

		suite.Cleanup()

		suite.Require().NoError(err)
	})
	t.Run("should list 2 assignees", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertMember(fixtures.GetMemberRow(p, USER_TWO, core.ROLE_MEMBER))
		suite.fixtures.InsertMember(fixtures.GetMemberRow(p, USER_THREE, core.ROLE_MEMBER))
		taskID := suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_UNASSIGNED))
		suite.fixtures.InsertAssignee(fixtures.GetAssigneeRow(p, taskID, USER_TWO))
		suite.fixtures.InsertAssignee(fixtures.GetAssigneeRow(p, taskID, USER_THREE))

		task, _ := suite.repo.Get(suite.ctx, taskID)

		suite.Cleanup()

		suite.Require().Equal(2, len(task.Assignees))
	})
	t.Run("should list 2 assignees with ID", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertMember(fixtures.GetMemberRow(p, USER_TWO, core.ROLE_MEMBER))
		suite.fixtures.InsertMember(fixtures.GetMemberRow(p, USER_THREE, core.ROLE_MEMBER))
		taskID := suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_UNASSIGNED))
		suite.fixtures.InsertAssignee(fixtures.GetAssigneeRow(p, taskID, USER_TWO))
		suite.fixtures.InsertAssignee(fixtures.GetAssigneeRow(p, taskID, USER_THREE))

		task, _ := suite.repo.Get(suite.ctx, taskID)

		suite.Cleanup()

		suite.ElementsMatch(
			[]string{USER_TWO, USER_THREE},
			[]string{task.Assignees[0].AssigneeID, task.Assignees[1].AssigneeID},
		)
	})
}

func (suite *taskRepositoryTestSuite) TestTaskList() {
	t := suite.T()

	t.Run("should get empty list", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))

		tasks, err := suite.repo.List(suite.ctx, p)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().NotNil(tasks)
		suite.Require().Equal(0, len(tasks))
	})
	t.Run("should get list of tasks", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_UNASSIGNED))

		_, err := suite.repo.List(suite.ctx, p)

		suite.Cleanup()

		suite.Require().NoError(err)
	})
	t.Run("should get list of 2 tasks", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_UNASSIGNED))
		suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_UNASSIGNED))

		tasks, _ := suite.repo.List(suite.ctx, p)

		suite.Cleanup()

		suite.Require().Equal(2, len(tasks))
	})
	t.Run("should get list of 2 tasks with IDs", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		t1 := suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_UNASSIGNED))
		t2 := suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_UNASSIGNED))

		tasks, _ := suite.repo.List(suite.ctx, p)

		suite.Cleanup()

		suite.Require().ElementsMatch(
			[]string{t1, t2},
			[]string{tasks[0].ID, tasks[1].ID},
		)
	})
	t.Run("should get 1 task with 2 assignees", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertMember(fixtures.GetMemberRow(p, USER_TWO, core.ROLE_MEMBER))
		suite.fixtures.InsertMember(fixtures.GetMemberRow(p, USER_THREE, core.ROLE_MEMBER))
		task := suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_UNASSIGNED))
		suite.fixtures.InsertAssignee(fixtures.GetAssigneeRow(p, task, USER_TWO))
		suite.fixtures.InsertAssignee(fixtures.GetAssigneeRow(p, task, USER_THREE))

		tasks, _ := suite.repo.List(suite.ctx, p)

		suite.Cleanup()

		suite.Require().ElementsMatch(
			[]string{USER_TWO, USER_THREE},
			[]string{tasks[0].Assignees[0].AssigneeID, tasks[0].Assignees[1].AssigneeID},
		)
	})
}

func (suite *taskRepositoryTestSuite) TestTaskUpdate() {
	t := suite.T()

	t.Run("should update task title", func(t *testing.T) {
		projectID := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		taskID := suite.fixtures.InsertTask(fixtures.RandomTaskRow(projectID, core.TASK_STATUS_UNASSIGNED))
		newTitle := "New Title"

		err := suite.repo.Update(suite.ctx, taskID, &newTitle, nil, nil)

		task, _ := gorm.G[models.Task](suite.db).Where("id = ?", taskID).First(suite.ctx)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().Equal(newTitle, task.Title)
	})
	t.Run("should update task description", func(t *testing.T) {
		projectID := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		taskID := suite.fixtures.InsertTask(fixtures.RandomTaskRow(projectID, core.TASK_STATUS_UNASSIGNED))
		newDescription := "New Description"

		err := suite.repo.Update(suite.ctx, taskID, nil, &newDescription, nil)

		task, _ := gorm.G[models.Task](suite.db).Where("id = ?", taskID).First(suite.ctx)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().Equal(newDescription, *task.Description)
	})
	t.Run("should update task status", func(t *testing.T) {
		projectID := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		taskID := suite.fixtures.InsertTask(fixtures.RandomTaskRow(projectID, core.TASK_STATUS_UNASSIGNED))
		newStatus := core.TASK_STATUS_COMPLETED

		err := suite.repo.Update(suite.ctx, taskID, nil, nil, &newStatus)

		task, _ := gorm.G[models.Task](suite.db).Where("id = ?", taskID).First(suite.ctx)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().Equal(newStatus, task.Status.String)
	})
}

func (suite *taskRepositoryTestSuite) TestTaskRecentlyAssigned() {
	t := suite.T()

	t.Run("should get empty list of recently assigned tasks", func(t *testing.T) {
		tasks, err := suite.repo.RecentlyAssigned(suite.ctx, USER_TWO, 10)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().NotNil(tasks)
		suite.Require().Equal(0, len(tasks))
	})
	t.Run("should get 2 recently assigned tasks", func(t *testing.T) {
		projectID := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		t1 := suite.fixtures.InsertTask(fixtures.RandomTaskRow(projectID, core.TASK_STATUS_UNASSIGNED))
		t2 := suite.fixtures.InsertTask(fixtures.RandomTaskRow(projectID, core.TASK_STATUS_UNASSIGNED))
		suite.fixtures.InsertMember(fixtures.GetMemberRow(projectID, USER_TWO, core.ROLE_MEMBER))
		suite.fixtures.InsertAssignee(fixtures.GetAssigneeRow(projectID, t1, USER_TWO))
		suite.fixtures.InsertAssignee(fixtures.GetAssigneeRow(projectID, t2, USER_TWO))

		tasks, err := suite.repo.RecentlyAssigned(suite.ctx, USER_TWO, 10)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().Equal(2, len(tasks))
	})
}

func (suite *taskRepositoryTestSuite) TestTaskRecentlyUnassigned() {
	t := suite.T()

	t.Run("should get empty list of recently unassigned tasks", func(t *testing.T) {
		tasks, err := suite.repo.RecentlyUnassigned(suite.ctx, USER_ONE, 10)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().NotNil(tasks)
		suite.Require().Equal(0, len(tasks))
	})
	t.Run("should get 2 recently unassigned tasks", func(t *testing.T) {
		projectID := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertTask(fixtures.RandomTaskRow(projectID, core.TASK_STATUS_UNASSIGNED))
		suite.fixtures.InsertTask(fixtures.RandomTaskRow(projectID, core.TASK_STATUS_UNASSIGNED))

		tasks, err := suite.repo.RecentlyUnassigned(suite.ctx, USER_ONE, 10)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().Equal(2, len(tasks))
	})
}

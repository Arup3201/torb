package tasks

import (
	"context"
	"log"
	"testing"

	"github.com/Arup3201/torb/core"
	"github.com/Arup3201/torb/core/assignees"
	"github.com/Arup3201/torb/core/members"
	"github.com/Arup3201/torb/models"
	"github.com/Arup3201/torb/testdata"
	"github.com/Arup3201/torb/testhelpers"
	"github.com/Arup3201/torb/testhelpers/fixtures"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type taskServiceTestSuite struct {
	suite.Suite
	ctx         context.Context
	pgContainer *testhelpers.PostgresContainer
	db          *gorm.DB
	fixtures    *fixtures.Fixtures
	service     *TaskService
}

func TestProjectService(t *testing.T) {
	suite.Run(t, new(taskServiceTestSuite))
}

func (suite *taskServiceTestSuite) SetupSuite() {
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

	taskRepo := NewTaskRepository(suite.db)
	memberRepo := members.NewMemberRepository(suite.db)
	assigneeRepo := assignees.NewAssigneeRepository(suite.db)
	suite.service = NewTaskService(taskRepo, memberRepo, assigneeRepo)

	suite.fixtures = fixtures.New(suite.ctx, suite.db)

	USER_ONE = suite.fixtures.InsertUser(fixtures.RandomUserRow())
	USER_TWO = suite.fixtures.InsertUser(fixtures.RandomUserRow())
	USER_THREE = suite.fixtures.InsertUser(fixtures.RandomUserRow())
}

func (suite *taskServiceTestSuite) Cleanup() {
	err := suite.db.WithContext(suite.ctx).
		Exec("DELETE FROM projects").Error
	suite.Require().NoError(err)
}

func (suite *taskServiceTestSuite) TestTaskCreate() {
	t := suite.T()

	t.Run("should create task", func(t *testing.T) {
		p := suite.fixtures.InsertProject(models.Project{
			Name:    "Project Fixture A",
			OwnerID: USER_ONE,
		})
		sample_title := "sample task"
		sample_description := "sample description"

		_, err := suite.service.Create(suite.ctx,
			p, USER_ONE,
			sample_title, sample_description, core.TASK_STATUS_UNASSIGNED)

		suite.Cleanup()

		suite.Require().NoError(err)
	})
	t.Run("should create task with unassigned status", func(t *testing.T) {
		p := suite.fixtures.InsertProject(models.Project{
			Name:    "Project Fixture A",
			OwnerID: USER_ONE,
		})
		sample_title := "sample task"
		sample_description := "sample description"

		taskId, _ := suite.service.Create(suite.ctx,
			p, USER_ONE,
			sample_title, sample_description, core.TASK_STATUS_UNASSIGNED)

		var status string
		suite.db.WithContext(suite.ctx).
			Table("tasks").
			Select("status").
			Where("id = ?", taskId).
			Scan(&status)

		suite.Cleanup()

		suite.Require().Equal(core.TASK_STATUS_UNASSIGNED, status)
	})
	t.Run("should be forbidden for non-owner", func(t *testing.T) {
		p := suite.fixtures.InsertProject(models.Project{
			Name:    "Project Fixture A",
			OwnerID: USER_ONE,
		})
		suite.fixtures.InsertMember(fixtures.GetMemberRow(p, USER_TWO, core.ROLE_MEMBER))
		sample_title := "sample task"
		sample_description := "sample description"

		_, err := suite.service.Create(suite.ctx,
			p, USER_TWO,
			sample_title, sample_description, core.TASK_STATUS_UNASSIGNED)

		suite.Cleanup()

		suite.Require().ErrorIs(err, core.ErrForbidden)
	})
	t.Run("should be invalid with empty task title", func(t *testing.T) {
		p := suite.fixtures.InsertProject(models.Project{
			Name:    "Project Fixture A",
			OwnerID: USER_ONE,
		})
		sample_title := ""
		sample_description := "sample description"

		_, err := suite.service.Create(suite.ctx,
			p, USER_ONE,
			sample_title, sample_description, core.TASK_STATUS_UNASSIGNED)

		suite.Cleanup()

		suite.Require().ErrorIs(err, core.ErrInvalidValue)
	})
	t.Run("should give error with invalid status", func(t *testing.T) {
		p := suite.fixtures.InsertProject(models.Project{
			Name:    "Project Fixture A",
			OwnerID: USER_ONE,
		})
		sample_title := "sample task"
		sample_description := "sample description"

		_, err := suite.service.Create(suite.ctx,
			p, USER_ONE,
			sample_title, sample_description, "UNKNOWN")

		suite.Cleanup()

		suite.Require().ErrorIs(err, core.ErrInvalidValue)
	})
}

func (suite *taskServiceTestSuite) TestTaskList() {
	t := suite.T()

	t.Run("should give empty list of tasks", func(t *testing.T) {
		p := suite.fixtures.InsertProject(models.Project{
			Name:    "Project Fixture A",
			OwnerID: USER_ONE,
		})

		tasks, err := suite.service.List(suite.ctx, p, USER_ONE)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().NotNil(tasks)
		suite.Require().Equal(0, len(tasks))
	})
	t.Run("should give 2 tasks", func(t *testing.T) {
		p := suite.fixtures.InsertProject(models.Project{
			Name:    "Project Fixture A",
			OwnerID: USER_ONE,
		})
		suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_ONGOING))
		suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_ONGOING))

		tasks, err := suite.service.List(suite.ctx, p, USER_ONE)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().Equal(2, len(tasks))
	})
	t.Run("should give 1 task with assignees", func(t *testing.T) {
		p := suite.fixtures.InsertProject(models.Project{
			Name:    "Project Fixture A",
			OwnerID: USER_ONE,
		})
		suite.fixtures.InsertMember(fixtures.GetMemberRow(p, USER_TWO, core.ROLE_MEMBER))
		taskID := suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_ONGOING))
		suite.fixtures.InsertAssignee(fixtures.GetAssigneeRow(p, taskID, USER_TWO))

		tasks, err := suite.service.List(suite.ctx, p, USER_ONE)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().ElementsMatch(
			[]string{USER_TWO},
			[]string{tasks[0].Assignees[0].UserID})
	})
}

func (suite *taskServiceTestSuite) TestTaskGet() {
	t := suite.T()

	t.Run("should get task", func(t *testing.T) {
		projectId := suite.fixtures.InsertProject(models.Project{
			Name:    "Project Fixture A",
			OwnerID: USER_ONE,
		})
		taskId := suite.fixtures.InsertTask(models.Task{
			Title:     "Project Task A",
			ProjectID: projectId,
			Status: models.TaskStatus{
				String: core.TASK_STATUS_UNASSIGNED,
			},
		})

		_, err := suite.service.Get(suite.ctx, projectId, taskId, USER_ONE)

		suite.Cleanup()

		suite.Require().NoError(err)
	})
	t.Run("should get task with id title status", func(t *testing.T) {
		projectId := suite.fixtures.InsertProject(models.Project{
			Name:    "Project Fixture A",
			OwnerID: USER_ONE,
		})
		sample_title := "sample task"
		taskId := suite.fixtures.InsertTask(models.Task{
			Title:     sample_title,
			ProjectID: projectId,
			Status: models.TaskStatus{
				String: core.TASK_STATUS_UNASSIGNED,
			},
		})

		task, _ := suite.service.Get(suite.ctx, projectId, taskId, USER_ONE)

		suite.Cleanup()

		suite.Require().Equal(taskId, task.ID)
		suite.Require().Equal(sample_title, task.Title)
		suite.Require().Equal(core.TASK_STATUS_UNASSIGNED, task.Status)
	})
	t.Run("should get task with description", func(t *testing.T) {
		projectId := suite.fixtures.InsertProject(models.Project{
			Name:    "Project Fixture A",
			OwnerID: USER_ONE,
		})
		sample_title := "sample task"
		sample_description := "Project description"
		taskId := suite.fixtures.InsertTask(models.Task{
			Title:       sample_title,
			ProjectID:   projectId,
			Description: &sample_description,
			Status: models.TaskStatus{
				String: core.TASK_STATUS_UNASSIGNED,
			},
		})

		task, _ := suite.service.Get(suite.ctx, projectId, taskId, USER_ONE)

		suite.Cleanup()

		suite.Require().Equal(sample_description, *task.Description)
	})
	t.Run("shoudl get task with assignees", func(t *testing.T) {
		p := suite.fixtures.InsertProject(models.Project{
			Name:    "Project Fixture A",
			OwnerID: USER_ONE,
		})
		suite.fixtures.InsertMember(fixtures.GetMemberRow(p, USER_TWO, core.ROLE_MEMBER))
		suite.fixtures.InsertMember(fixtures.GetMemberRow(p, USER_THREE, core.ROLE_MEMBER))
		taskID := suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_ONGOING))
		suite.fixtures.InsertAssignee(fixtures.GetAssigneeRow(p, taskID, USER_TWO))
		suite.fixtures.InsertAssignee(fixtures.GetAssigneeRow(p, taskID, USER_THREE))

		task, err := suite.service.Get(suite.ctx, p, taskID, USER_ONE)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().ElementsMatch(
			[]string{USER_TWO, USER_THREE},
			[]string{task.Assignees[0].UserID, task.Assignees[1].UserID},
		)
	})
	t.Run("should be forbidden for normal user", func(t *testing.T) {
		projectId := suite.fixtures.InsertProject(models.Project{
			Name:    "Project Fixture A",
			OwnerID: USER_ONE,
		})
		sample_title := "sample task"
		taskId := suite.fixtures.InsertTask(models.Task{
			Title:     sample_title,
			ProjectID: projectId,
			Status: models.TaskStatus{
				String: core.TASK_STATUS_UNASSIGNED,
			},
		})

		_, err := suite.service.Get(suite.ctx, projectId, taskId, USER_TWO)

		suite.Cleanup()

		suite.Require().ErrorIs(err, core.ErrForbidden)
	})
}

func (suite *taskServiceTestSuite) TestTaskUpdate() {
	t := suite.T()

	t.Run("should update task", func(t *testing.T) {
		projectId := suite.fixtures.InsertProject(models.Project{
			Name:    "Project Fixture A",
			OwnerID: USER_ONE,
		})
		taskId := suite.fixtures.InsertTask(fixtures.RandomTaskRow(projectId, core.TASK_STATUS_ONGOING))
		updatedTaskTitle := "Project title updated"

		err := suite.service.Update(suite.ctx,
			projectId, taskId, USER_ONE,
			&updatedTaskTitle, nil, nil)

		suite.Cleanup()

		suite.Require().NoError(err)
	})
	t.Run("should update task with title", func(t *testing.T) {
		projectId := suite.fixtures.InsertProject(models.Project{
			Name:    "Project Fixture A",
			OwnerID: USER_ONE,
		})
		taskId := suite.fixtures.InsertTask(fixtures.RandomTaskRow(projectId, core.TASK_STATUS_ONGOING))

		updatedTaskTitle := "Project title updated"

		suite.service.Update(suite.ctx,
			projectId, taskId, USER_ONE,
			&updatedTaskTitle, nil, nil)

		var title string
		suite.db.WithContext(suite.ctx).
			Table("tasks").
			Select("title").
			Where("id = ?", taskId).
			Scan(&title)

		suite.Cleanup()

		suite.Require().Equal(updatedTaskTitle, title)
	})
	t.Run("should update description and status", func(t *testing.T) {
		projectId := suite.fixtures.InsertProject(models.Project{
			Name:    "Project Fixture A",
			OwnerID: USER_ONE,
		})
		taskId := suite.fixtures.InsertTask(fixtures.RandomTaskRow(projectId, core.TASK_STATUS_ONGOING))
		updatedTaskDesc := "Project description updated"
		updatedStatus := core.TASK_STATUS_ABANDONED

		suite.service.Update(suite.ctx,
			projectId, taskId, USER_ONE,
			nil, &updatedTaskDesc, &updatedStatus)

		task, _ := gorm.G[models.Task](suite.db).Where("id = ?", taskId).First(suite.ctx)

		suite.Cleanup()

		suite.Require().NotNil(task.Description)
		suite.Require().Equal(updatedTaskDesc, *task.Description)
		suite.Require().Equal(core.TASK_STATUS_ABANDONED, task.Status.String)
	})
	t.Run("should be invalid to update status with invalid value", func(t *testing.T) {
		projectId := suite.fixtures.InsertProject(models.Project{
			Name:    "Project Fixture A",
			OwnerID: USER_ONE,
		})
		taskId := suite.fixtures.InsertTask(fixtures.RandomTaskRow(projectId, core.TASK_STATUS_ONGOING))
		updatedStatus := "Unknown"

		err := suite.service.Update(suite.ctx,
			projectId, taskId, USER_ONE,
			nil, nil, &updatedStatus)

		suite.Cleanup()

		suite.Require().ErrorIs(err, core.ErrInvalidValue)
	})
	t.Run("should be invalid with all nil", func(t *testing.T) {
		projectId := suite.fixtures.InsertProject(models.Project{
			Name:    "Project Fixture A",
			OwnerID: USER_ONE,
		})
		taskId := suite.fixtures.InsertTask(fixtures.RandomTaskRow(projectId, core.TASK_STATUS_ONGOING))

		err := suite.service.Update(suite.ctx,
			projectId, taskId, USER_ONE,
			nil, nil, nil)

		suite.Cleanup()

		suite.Require().ErrorIs(err, core.ErrInvalidValue)
	})
	t.Run("should be forbidden to update title as member", func(t *testing.T) {
		projectId := suite.fixtures.InsertProject(models.Project{
			Name:    "Project Fixture A",
			OwnerID: USER_ONE,
		})
		taskId := suite.fixtures.InsertTask(fixtures.RandomTaskRow(projectId, core.TASK_STATUS_ONGOING))
		suite.fixtures.InsertMember(fixtures.GetMemberRow(projectId, USER_TWO, core.ROLE_MEMBER))
		updatedTaskTitle := "Project title updated"

		err := suite.service.Update(suite.ctx,
			projectId, taskId, USER_TWO,
			&updatedTaskTitle, nil, nil)

		suite.Cleanup()

		suite.Require().Error(err, core.ErrForbidden)
	})
	t.Run("should be able to update title as assignee", func(t *testing.T) {
		projectId := suite.fixtures.InsertProject(models.Project{
			Name:    "Project Fixture A",
			OwnerID: USER_ONE,
		})
		suite.fixtures.InsertMember(fixtures.GetMemberRow(projectId, USER_TWO, core.ROLE_MEMBER))
		taskId := suite.fixtures.InsertTask(fixtures.RandomTaskRow(projectId, core.TASK_STATUS_ONGOING))
		suite.fixtures.InsertAssignee(fixtures.GetAssigneeRow(projectId, taskId, USER_TWO))
		updatedTaskTitle := "Project title updated"

		err := suite.service.Update(suite.ctx,
			projectId, taskId, USER_TWO,
			&updatedTaskTitle, nil, nil)

		suite.Cleanup()

		suite.Require().NoError(err)
	})
}

func (suite *taskServiceTestSuite) TestRecentlyAssigned() {
	t := suite.T()

	t.Run("should get empty list for recently assigned tasks", func(t *testing.T) {
		projectId := suite.fixtures.InsertProject(models.Project{
			Name:    "Project Fixture A",
			OwnerID: USER_ONE,
		})
		suite.fixtures.InsertMember(fixtures.GetMemberRow(projectId, USER_TWO, core.ROLE_MEMBER))
		suite.fixtures.InsertTask(fixtures.RandomTaskRow(projectId, core.TASK_STATUS_ONGOING))

		tasks, err := suite.service.RecentlyAssigned(suite.ctx, USER_ONE)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().NotNil(tasks)
		suite.Require().Equal(0, len(tasks))
	})
	t.Run("should get project name with recently assigned task", func(t *testing.T) {
		name := "Project Fixture A"
		projectId := suite.fixtures.InsertProject(models.Project{
			Name:    name,
			OwnerID: USER_ONE,
		})
		suite.fixtures.InsertMember(fixtures.GetMemberRow(projectId, USER_TWO, core.ROLE_MEMBER))
		taskId := suite.fixtures.InsertTask(fixtures.RandomTaskRow(projectId, core.TASK_STATUS_ONGOING))
		suite.fixtures.InsertAssignee(fixtures.GetAssigneeRow(projectId, taskId, USER_TWO))

		tasks, err := suite.service.RecentlyAssigned(suite.ctx, USER_TWO)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().Equal(1, len(tasks))
		suite.Require().Equal(name, tasks[0].ProjectName)
	})
}

func (suite *taskServiceTestSuite) TestRecentlyUnassigned() {
	t := suite.T()

	t.Run("should get empty list for recently unassigned tasks", func(t *testing.T) {
		tasks, err := suite.service.RecentlyUnassigned(suite.ctx, USER_ONE)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().NotNil(tasks)
		suite.Require().Equal(0, len(tasks))
	})
	t.Run("should get project name with recently unassigned task", func(t *testing.T) {
		name := "Project Fixture A"
		projectId := suite.fixtures.InsertProject(models.Project{
			Name:    name,
			OwnerID: USER_ONE,
		})
		suite.fixtures.InsertTask(fixtures.RandomTaskRow(projectId, core.TASK_STATUS_UNASSIGNED))

		tasks, err := suite.service.RecentlyUnassigned(suite.ctx, USER_ONE)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().Equal(1, len(tasks))
		suite.Require().Equal(name, tasks[0].ProjectName)
	})
}

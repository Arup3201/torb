package assignees

import (
	"context"
	"log"
	"testing"

	"github.com/Arup3201/torb/core"
	"github.com/Arup3201/torb/core/members"
	"github.com/Arup3201/torb/testdata"
	"github.com/Arup3201/torb/testhelpers"
	"github.com/Arup3201/torb/testhelpers/fixtures"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type assigneeServiceTestSuite struct {
	suite.Suite
	ctx         context.Context
	pgContainer *testhelpers.PostgresContainer
	db          *gorm.DB
	fixtures    *fixtures.Fixtures
	service     *AssigneeService
}

func TestAssigneeService(t *testing.T) {
	suite.Run(t, new(assigneeServiceTestSuite))
}

func (suite *assigneeServiceTestSuite) SetupSuite() {
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

	memberRepo := members.NewMemberRepository(suite.db)
	assigneeRepo := NewAssigneeRepository(suite.db)
	service := NewAssigneeService(memberRepo, assigneeRepo)
	suite.service = service

	suite.fixtures = fixtures.New(suite.ctx, suite.db)

	USER_ONE = suite.fixtures.InsertUser(fixtures.RandomUserRow())
	USER_TWO = suite.fixtures.InsertUser(fixtures.RandomUserRow())
}

func (suite *assigneeServiceTestSuite) Cleanup() {
	err := suite.db.WithContext(suite.ctx).
		Exec("DELETE FROM projects").Error
	suite.Require().NoError(err)
}

func (suite *assigneeServiceTestSuite) TestAddAssignee() {
	t := suite.T()

	t.Run("should add assignee when owner requests", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		task := suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_ONGOING))

		err := suite.service.AddAssignee(suite.ctx, p, task, USER_ONE, USER_TWO)

		suite.Cleanup()

		suite.Require().NoError(err)
	})

	t.Run("should return duplicate when assignee already exists", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		task := suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_ONGOING))
		suite.fixtures.InsertAssignee(fixtures.GetAssigneeRow(p, task, USER_TWO))

		err := suite.service.AddAssignee(suite.ctx, p, task, USER_ONE, USER_TWO)

		suite.Cleanup()

		suite.Require().ErrorIs(err, core.ErrDuplicate)
	})

	t.Run("should return forbidden when requester is not owner", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		task := suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_ONGOING))
		// USER_TWO is not owner

		err := suite.service.AddAssignee(suite.ctx, p, task, USER_TWO, USER_ONE)

		suite.Cleanup()

		suite.Require().ErrorIs(err, core.ErrForbidden)
	})
}

func (suite *assigneeServiceTestSuite) TestRemoveAssignee() {
	t := suite.T()

	t.Run("should remove assignee when owner requests", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		task := suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_ONGOING))
		suite.fixtures.InsertAssignee(fixtures.GetAssigneeRow(p, task, USER_TWO))

		err := suite.service.RemoveAssignee(suite.ctx, p, task, USER_ONE, USER_TWO)

		suite.Cleanup()

		suite.Require().NoError(err)
	})

	t.Run("should return not found when assignee does not exist", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		task := suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_ONGOING))

		err := suite.service.RemoveAssignee(suite.ctx, p, task, USER_ONE, USER_TWO)

		suite.Cleanup()

		suite.Require().ErrorIs(err, core.ErrNotFound)
	})

	t.Run("should return forbidden when requester is not owner", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		task := suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_ONGOING))
		suite.fixtures.InsertAssignee(fixtures.GetAssigneeRow(p, task, USER_TWO))

		err := suite.service.RemoveAssignee(suite.ctx, p, task, USER_TWO, USER_TWO)

		suite.Cleanup()

		suite.Require().ErrorIs(err, core.ErrForbidden)
	})
}

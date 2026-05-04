package assignees

import (
	"context"
	"log"
	"testing"

	"github.com/Arup3201/torb/core"
	"github.com/Arup3201/torb/testdata"
	"github.com/Arup3201/torb/testhelpers"
	"github.com/Arup3201/torb/testhelpers/fixtures"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var USER_ONE, USER_TWO string

type assigneeRepositoryTestSuite struct {
	suite.Suite
	pgContainer *testhelpers.PostgresContainer
	db          *gorm.DB
	fixtures    *fixtures.Fixtures
	repo        *AssigneeRepository
	ctx         context.Context
}

func (suite *assigneeRepositoryTestSuite) SetupSuite() {
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

	suite.repo = NewAssigneeRepository(suite.db)

	err = testdata.TestMigrate(suite.db)
	if err != nil {
		log.Fatal(err)
	}

	suite.fixtures = fixtures.New(suite.ctx, suite.db)

	USER_ONE = suite.fixtures.InsertUser(fixtures.RandomUserRow())
	USER_TWO = suite.fixtures.InsertUser(fixtures.RandomUserRow())
}

func (suite *assigneeRepositoryTestSuite) Cleanup() {
	err := suite.db.WithContext(suite.ctx).
		Exec("DELETE FROM projects").Error
	suite.Require().NoError(err)
}

func TestAssigneeRepository(t *testing.T) {
	suite.Run(t, new(assigneeRepositoryTestSuite))
}

func (suite *assigneeRepositoryTestSuite) TestCreate() {
	t := suite.T()

	t.Run("should create assignee", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		task := suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_ONGOING))

		err := suite.repo.Create(suite.ctx, p, task, USER_TWO)

		suite.Cleanup()

		suite.Require().NoError(err)
	})
}

func (suite *assigneeRepositoryTestSuite) TestIs() {
	t := suite.T()

	t.Run("should return not found when none exists", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		task := suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_ONGOING))

		err := suite.repo.Is(suite.ctx, p, task, USER_TWO)

		suite.Cleanup()

		suite.Require().ErrorIs(err, core.ErrNotFound)
	})

	t.Run("should return nil when assignee exists", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		task := suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_ONGOING))
		suite.fixtures.InsertAssignee(fixtures.GetAssigneeRow(p, task, USER_TWO))

		err := suite.repo.Is(suite.ctx, p, task, USER_TWO)

		suite.Cleanup()

		suite.Require().NoError(err)
	})
}

func (suite *assigneeRepositoryTestSuite) TestDelete() {
	t := suite.T()

	t.Run("should delete assignee", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		task := suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_ONGOING))
		suite.fixtures.InsertAssignee(fixtures.GetAssigneeRow(p, task, USER_TWO))

		err := suite.repo.Delete(suite.ctx, p, task, USER_TWO)

		suite.Cleanup()

		suite.Require().NoError(err)
	})

	t.Run("should not error when deleting non existing assignee", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		task := suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_ONGOING))

		err := suite.repo.Delete(suite.ctx, p, task, USER_TWO)

		suite.Cleanup()

		suite.Require().NoError(err)
	})
}

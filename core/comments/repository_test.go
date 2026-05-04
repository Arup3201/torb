package comments

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

type commentRepositoryTestSuite struct {
	suite.Suite
	pgContainer *testhelpers.PostgresContainer
	db          *gorm.DB
	fixtures    *fixtures.Fixtures
	repo        *CommentRepository
	ctx         context.Context
}

func (suite *commentRepositoryTestSuite) SetupSuite() {
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

	suite.repo = NewCommentRepository(suite.db)

	err = testdata.TestMigrate(suite.db)
	if err != nil {
		log.Fatal(err)
	}

	suite.fixtures = fixtures.New(suite.ctx, suite.db)

	USER_ONE = suite.fixtures.InsertUser(fixtures.RandomUserRow())
	USER_TWO = suite.fixtures.InsertUser(fixtures.RandomUserRow())
}

func (suite *commentRepositoryTestSuite) Cleanup() {
	err := suite.db.WithContext(suite.ctx).
		Exec("DELETE FROM projects").Error
	suite.Require().NoError(err)
}

func TestCommentRepository(t *testing.T) {
	suite.Run(t, new(commentRepositoryTestSuite))
}

func (suite *commentRepositoryTestSuite) TestCreate() {
	t := suite.T()

	t.Run("should create comment", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		task := suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_ONGOING))

		id, err := suite.repo.Create(suite.ctx, p, task, USER_ONE, "hello")

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().NotEmpty(id)
	})
}

func (suite *commentRepositoryTestSuite) TestList() {
	t := suite.T()

	t.Run("should return empty list", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		task := suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_ONGOING))

		rows, err := suite.repo.List(suite.ctx, p, task)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().NotNil(rows)
		suite.Require().Equal(0, len(rows))
	})

	t.Run("should return 2 comments", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		task := suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_ONGOING))
		suite.fixtures.InsertComment(fixtures.GetCommentRow(p, task, USER_ONE, "hello"))
		suite.fixtures.InsertComment(fixtures.GetCommentRow(p, task, USER_TWO, "world"))

		rows, err := suite.repo.List(suite.ctx, p, task)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().Equal(2, len(rows))
	})

	t.Run("should return 2 comments with correct values", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		task := suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_ONGOING))
		suite.fixtures.InsertComment(fixtures.GetCommentRow(p, task, USER_ONE, "hello"))
		suite.fixtures.InsertComment(fixtures.GetCommentRow(p, task, USER_TWO, "world"))

		rows, _ := suite.repo.List(suite.ctx, p, task)

		suite.Cleanup()

		suite.Require().ElementsMatch(
			[]string{USER_ONE, USER_TWO},
			[]string{rows[0].UserID, rows[1].UserID},
		)
	})
}

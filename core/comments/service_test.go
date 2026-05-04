package comments

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

var USER_THREE string

type commentServiceTestSuite struct {
	suite.Suite
	ctx         context.Context
	pgContainer *testhelpers.PostgresContainer
	db          *gorm.DB
	fixtures    *fixtures.Fixtures
	service     *CommentService
}

func TestCommentService(t *testing.T) {
	suite.Run(t, new(commentServiceTestSuite))
}

func (suite *commentServiceTestSuite) SetupSuite() {
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

	commentRepo := NewCommentRepository(suite.db)
	memberRepo := members.NewMemberRepository(suite.db)
	service := NewCommentService(commentRepo, memberRepo)
	suite.service = service

	suite.fixtures = fixtures.New(suite.ctx, suite.db)

	USER_ONE = suite.fixtures.InsertUser(fixtures.RandomUserRow())
	USER_TWO = suite.fixtures.InsertUser(fixtures.RandomUserRow())
	USER_THREE = suite.fixtures.InsertUser(fixtures.RandomUserRow())
}

func (suite *commentServiceTestSuite) Cleanup() {
	err := suite.db.WithContext(suite.ctx).
		Exec("DELETE FROM projects").Error
	suite.Require().NoError(err)
}

func (suite *commentServiceTestSuite) TestCreate() {
	t := suite.T()

	t.Run("should create comment when member", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		task := suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_ONGOING))

		id, err := suite.service.Create(suite.ctx, p, task, USER_ONE, "hello")

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().NotEmpty(id)
	})

	t.Run("should return invalid value for empty comment", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		task := suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_ONGOING))

		_, err := suite.service.Create(suite.ctx, p, task, USER_ONE, "    ")

		suite.Cleanup()

		suite.Require().ErrorIs(err, core.ErrInvalidValue)
	})

	t.Run("should return forbidden when requester is not a member", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		task := suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_ONGOING))

		_, err := suite.service.Create(suite.ctx, p, task, USER_TWO, "hi")

		suite.Cleanup()

		suite.Require().ErrorIs(err, core.ErrForbidden)
	})
}

func (suite *commentServiceTestSuite) TestList() {
	t := suite.T()

	t.Run("should return empty list when no comments", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		task := suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_ONGOING))

		rows, err := suite.service.List(suite.ctx, p, task, USER_ONE)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().NotNil(rows)
		suite.Require().Equal(0, len(rows))
	})

	t.Run("should return comments for member", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		task := suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_ONGOING))
		suite.fixtures.InsertComment(fixtures.GetCommentRow(p, task, USER_ONE, "hello"))
		suite.fixtures.InsertComment(fixtures.GetCommentRow(p, task, USER_TWO, "world"))

		rows, err := suite.service.List(suite.ctx, p, task, USER_ONE)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().Equal(2, len(rows))
	})

	t.Run("should return forbidden when requester is not a member", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		task := suite.fixtures.InsertTask(fixtures.RandomTaskRow(p, core.TASK_STATUS_ONGOING))
		suite.fixtures.InsertComment(fixtures.GetCommentRow(p, task, USER_ONE, "hello"))

		_, err := suite.service.List(suite.ctx, p, task, USER_THREE)

		suite.Cleanup()

		suite.Require().ErrorIs(err, core.ErrForbidden)
	})
}

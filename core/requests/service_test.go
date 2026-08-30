package requests

import (
	"context"
	"log"
	"testing"

	"github.com/Arup3201/torb/core"
	"github.com/Arup3201/torb/core/documents"
	"github.com/Arup3201/torb/core/members"
	"github.com/Arup3201/torb/models"
	"github.com/Arup3201/torb/testdata"
	"github.com/Arup3201/torb/testhelpers"
	"github.com/Arup3201/torb/testhelpers/fixtures"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type joinServiceTestSuite struct {
	suite.Suite
	ctx         context.Context
	pgContainer *testhelpers.PostgresContainer
	db          *gorm.DB
	fixtures    *fixtures.Fixtures
	service     *JoinRequestService
}

func TestProjectService(t *testing.T) {
	suite.Run(t, new(joinServiceTestSuite))
}

func (suite *joinServiceTestSuite) SetupSuite() {
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

	txManager := core.NewTxManager(suite.db)
	joinRepo := NewJoinRepository(suite.db)
	memberRepo := members.NewMemberRepository(suite.db)

	cfg, err := config.LoadDefaultConfig(suite.ctx)
	suite.NoError(err)
	s3Client := s3.NewFromConfig(cfg)
	docStorage := documents.NewDocumentStorage(s3Client)

	suite.service = NewJoinRequestService(txManager, joinRepo, memberRepo, docStorage)

	suite.fixtures = fixtures.New(suite.ctx, suite.db)

	USER_ONE = suite.fixtures.InsertUser(fixtures.RandomUserRow())
	USER_TWO = suite.fixtures.InsertUser(fixtures.RandomUserRow())
	USER_THREE = suite.fixtures.InsertUser(fixtures.RandomUserRow())
}

func (suite *joinServiceTestSuite) Cleanup() {
	err := suite.db.WithContext(suite.ctx).
		Exec("DELETE FROM projects").Error
	suite.Require().NoError(err)
}

func (suite *joinServiceTestSuite) TestCreate() {

	t := suite.T()

	t.Run("should create join request with Pending status", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))

		err := suite.service.Create(suite.ctx, p, USER_TWO)

		var status string
		gorm.G[models.JoinRequest](suite.db).Select("status").Where("project_id = ? AND user_id = ?", p, USER_TWO).Scan(suite.ctx, &status)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().Equal(core.JOIN_STATUS_PENDING, status)
	})
}

func (suite *joinServiceTestSuite) TestStatus() {
	t := suite.T()

	t.Run("should get join request status as Accepted", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertJoinRequest(models.JoinRequest{
			ProjectID: p,
			UserID:    USER_TWO,
			Status:    models.JoinStatus{String: core.JOIN_STATUS_ACCEPTED},
		})

		status, err := suite.service.GetStatus(suite.ctx, p, USER_TWO)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().Equal(core.JOIN_STATUS_ACCEPTED, status)
	})
}

func (suite *joinServiceTestSuite) TestList() {
	t := suite.T()

	t.Run("should give empty list of join requests", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))

		joins, err := suite.service.List(suite.ctx, p, USER_ONE)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().NotNil(joins)
		suite.Require().Equal(0, len(joins))
	})
	t.Run("should get 2 join requests", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertJoinRequest(fixtures.GetJoinRequest(p, USER_TWO))
		suite.fixtures.InsertJoinRequest(fixtures.GetJoinRequest(p, USER_THREE))

		joins, err := suite.service.List(suite.ctx, p, USER_ONE)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().Equal(2, len(joins))
	})
	t.Run("should get forbidden error", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertJoinRequest(fixtures.GetJoinRequest(p, USER_TWO))
		suite.fixtures.InsertJoinRequest(fixtures.GetJoinRequest(p, USER_THREE))

		_, err := suite.service.List(suite.ctx, p, USER_TWO)

		suite.Cleanup()

		suite.Require().Error(err, core.ErrForbidden)
	})
}

func (suite *joinServiceTestSuite) TestRespond() {
	t := suite.T()

	t.Run("should respond to join request by accepting", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertJoinRequest(fixtures.GetJoinRequest(p, USER_TWO))

		err := suite.service.Respond(suite.ctx, p, USER_ONE, USER_TWO, core.JOIN_STATUS_ACCEPTED)

		suite.Cleanup()

		suite.Require().NoError(err)
	})
	t.Run("should create member when join request is accepted", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertJoinRequest(fixtures.GetJoinRequest(p, USER_TWO))

		suite.service.Respond(suite.ctx, p, USER_ONE, USER_TWO, core.JOIN_STATUS_ACCEPTED)

		var role string
		gorm.G[models.Member](suite.db).Select("role").Where("project_id = ? AND user_id = ?", p, USER_TWO).Scan(suite.ctx, &role)

		suite.Cleanup()

		suite.Require().Equal(core.ROLE_MEMBER, role)
	})
	t.Run("should respond to join request by rejecting", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertJoinRequest(fixtures.GetJoinRequest(p, USER_TWO))

		err := suite.service.Respond(suite.ctx, p, USER_ONE, USER_TWO, core.JOIN_STATUS_REJECTED)

		suite.Cleanup()

		suite.Require().NoError(err)
	})
	t.Run("should not create member when join request is rejected", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertJoinRequest(fixtures.GetJoinRequest(p, USER_TWO))

		suite.service.Respond(suite.ctx, p, USER_ONE, USER_TWO, core.JOIN_STATUS_REJECTED)

		_, err := gorm.G[models.Member](suite.db).Where("project_id = ? AND user_id = ?", p, USER_TWO).First(suite.ctx)

		suite.Cleanup()

		suite.Require().Error(err)
	})
	t.Run("should get invalid value error when join request status does not change", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertJoinRequest(fixtures.GetJoinRequest(p, USER_TWO))

		err := suite.service.Respond(suite.ctx, p, USER_ONE, USER_TWO, core.JOIN_STATUS_PENDING)

		suite.Cleanup()

		suite.Require().ErrorIs(err, core.ErrInvalidValue)
	})
	t.Run("should get invalid value error when status tried to change to pending from accepted", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertJoinRequest(models.JoinRequest{
			ProjectID: p,
			UserID:    USER_TWO,
			Status:    models.JoinStatus{String: core.JOIN_STATUS_ACCEPTED},
		})

		err := suite.service.Respond(suite.ctx, p, USER_ONE, USER_TWO, core.JOIN_STATUS_PENDING)

		suite.Cleanup()

		suite.Require().ErrorIs(err, core.ErrInvalidValue)
	})
	t.Run("should get invalid value error when status tried to change to rejected from accepted", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertJoinRequest(models.JoinRequest{
			ProjectID: p,
			UserID:    USER_TWO,
			Status:    models.JoinStatus{String: core.JOIN_STATUS_ACCEPTED},
		})

		err := suite.service.Respond(suite.ctx, p, USER_ONE, USER_TWO, core.JOIN_STATUS_REJECTED)

		suite.Cleanup()

		suite.Require().ErrorIs(err, core.ErrInvalidValue)
	})
}

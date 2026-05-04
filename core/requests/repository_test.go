package requests

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

type joinRepositoryTestSuite struct {
	suite.Suite
	pgContainer *testhelpers.PostgresContainer
	db          *gorm.DB
	fixtures    *fixtures.Fixtures
	repo        *JoinRepository
	ctx         context.Context
}

func (suite *joinRepositoryTestSuite) SetupSuite() {
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

	suite.repo = NewJoinRepository(suite.db)
	suite.fixtures = fixtures.New(suite.ctx, suite.db)

	USER_ONE = suite.fixtures.InsertUser(fixtures.RandomUserRow())
	USER_TWO = suite.fixtures.InsertUser(fixtures.RandomUserRow())
	USER_THREE = suite.fixtures.InsertUser(fixtures.RandomUserRow())
}

func (suite *joinRepositoryTestSuite) Cleanup() {
	err := suite.db.WithContext(suite.ctx).
		Exec("DELETE FROM projects").Error
	suite.Require().NoError(err)
}

func TestJoinRepository(t *testing.T) {
	suite.Run(t, new(joinRepositoryTestSuite))
}

func (suite *joinRepositoryTestSuite) TestCreate() {
	t := suite.T()

	t.Run("should create join request", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))

		err := suite.repo.Create(suite.ctx, p, USER_TWO, core.JOIN_STATUS_PENDING)

		suite.Cleanup()

		suite.Require().NoError(err)
	})
	t.Run("should create join request with Pending status", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))

		suite.repo.Create(suite.ctx, p, USER_TWO, core.JOIN_STATUS_PENDING)

		var status string
		gorm.G[models.JoinRequest](suite.db).Select("status").Where("project_id = ? AND user_id = ?", p, USER_TWO).Scan(suite.ctx, &status)

		suite.Cleanup()

		suite.Require().Equal(core.JOIN_STATUS_PENDING, status)
	})
	t.Run("should not create join request with invalid status", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))

		err := suite.repo.Create(suite.ctx, p, USER_TWO, "UNKNOWN")

		suite.Cleanup()

		suite.Require().Error(err)
	})
}

func (suite *joinRepositoryTestSuite) TestList() {
	t := suite.T()

	t.Run("should give empty list of join requests", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))

		joins, err := suite.repo.List(suite.ctx, p)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().NotNil(joins)
		suite.Require().Equal(0, len(joins))
	})
	t.Run("should give 2 join requests", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertJoinRequest(fixtures.GetJoinRequest(p, USER_TWO))
		suite.fixtures.InsertJoinRequest(fixtures.GetJoinRequest(p, USER_THREE))

		joins, err := suite.repo.List(suite.ctx, p)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().Equal(2, len(joins))
	})
	t.Run("should give 2 join requests with correct values", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertJoinRequest(fixtures.GetJoinRequest(p, USER_TWO))
		suite.fixtures.InsertJoinRequest(fixtures.GetJoinRequest(p, USER_THREE))

		joins, _ := suite.repo.List(suite.ctx, p)

		suite.Cleanup()

		suite.Require().ElementsMatch(
			[]string{USER_TWO, USER_THREE},
			[]string{joins[0].UserID, joins[1].UserID},
		)
	})
}

func (suite *joinRepositoryTestSuite) TestStatus() {
	t := suite.T()

	t.Run("should get Pending status", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertJoinRequest(fixtures.GetJoinRequest(p, USER_TWO))

		status, _ := suite.repo.Status(suite.ctx, p, USER_TWO)

		suite.Cleanup()

		suite.Require().Equal(core.JOIN_STATUS_PENDING, status)
	})
	t.Run("should get Accepted status", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertJoinRequest(models.JoinRequest{
			ProjectID: p,
			UserID:    USER_TWO,
			Status:    models.JoinStatus{String: core.JOIN_STATUS_ACCEPTED},
		})

		status, _ := suite.repo.Status(suite.ctx, p, USER_TWO)

		suite.Cleanup()

		suite.Require().Equal(core.JOIN_STATUS_ACCEPTED, status)
	})
	t.Run("should get not found error", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))

		_, err := suite.repo.Status(suite.ctx, p, USER_TWO)

		suite.Cleanup()

		suite.Require().ErrorIs(err, core.ErrNotFound)
	})
}

func (suite *joinRepositoryTestSuite) TestUpdate() {
	t := suite.T()

	t.Run("should update join request", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertJoinRequest(fixtures.GetJoinRequest(p, USER_TWO))

		err := suite.repo.Update(suite.ctx, p, USER_TWO, core.JOIN_STATUS_ACCEPTED)

		suite.Cleanup()

		suite.Require().NoError(err)
	})
	t.Run("should update join request with status Accepted", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertJoinRequest(fixtures.GetJoinRequest(p, USER_TWO))

		suite.repo.Update(suite.ctx, p, USER_TWO, core.JOIN_STATUS_ACCEPTED)

		var status string
		gorm.G[models.JoinRequest](suite.db).Select("status").Where("project_id = ? AND user_id = ?", p, USER_TWO).Scan(suite.ctx, &status)

		suite.Cleanup()

		suite.Require().Equal(core.JOIN_STATUS_ACCEPTED, status)
	})
	t.Run("should get not found error", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))

		err := suite.repo.Update(suite.ctx, p, USER_TWO, core.JOIN_STATUS_ACCEPTED)

		suite.Cleanup()

		suite.Require().ErrorIs(err, core.ErrNotFound)
	})
	t.Run("should get invalid join status error", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertJoinRequest(fixtures.GetJoinRequest(p, USER_TWO))

		err := suite.repo.Update(suite.ctx, p, USER_TWO, "UNKNOWN")

		suite.Cleanup()

		suite.Require().Error(err)
	})
}

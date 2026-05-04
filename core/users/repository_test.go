package users

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

type userRepositoryTestSuite struct {
	suite.Suite
	pgContainer *testhelpers.PostgresContainer
	db          *gorm.DB
	fixtures    *fixtures.Fixtures
	repo        *UserRepository
	ctx         context.Context
}

func (suite *userRepositoryTestSuite) SetupSuite() {
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

	suite.repo = NewUserRepository(suite.db)

	err = testdata.TestMigrate(suite.db)
	if err != nil {
		log.Fatal(err)
	}

	suite.fixtures = fixtures.New(suite.ctx, suite.db)
}

func (suite *userRepositoryTestSuite) Cleanup() {
	err := suite.db.WithContext(suite.ctx).
		Exec("DELETE FROM users").Error
	suite.Require().NoError(err)
}

func TestUserRepository(t *testing.T) {
	suite.Run(t, new(userRepositoryTestSuite))
}

func (suite *userRepositoryTestSuite) TestCreate() {
	t := suite.T()

	t.Run("should create user", func(t *testing.T) {
		id, err := suite.repo.Create(suite.ctx, "alice", "alice@test.com", nil, nil)

		suite.Require().NoError(err)
		suite.Require().NotEmpty(id)
	})

	t.Run("should create user with display name and avatar url", func(t *testing.T) {
		dn := "Alice"
		au := "http://avatar"
		id, err := suite.repo.Create(suite.ctx, "alice2", "alice2@test.com", &dn, &au)

		suite.Require().NoError(err)
		suite.Require().NotEmpty(id)

		u, err := gorm.G[models.User](suite.db).Where("id = ?", id).First(suite.ctx)
		suite.Require().NoError(err)
		suite.Require().Equal(dn, *u.DisplayName)
		suite.Require().Equal(au, *u.AvatarURL)
	})
	t.Run("should get duplicate value error with same username", func(t *testing.T) {
		suite.repo.Create(suite.ctx, "alice3", "alice3@test.com", nil, nil)
		_, err := suite.repo.Create(suite.ctx, "alice3", "alicex@test.com", nil, nil)

		suite.Require().ErrorIs(err, core.ErrDuplicate)
	})
	t.Run("should get duplicate value error with same email", func(t *testing.T) {
		suite.repo.Create(suite.ctx, "alice4", "alice4@test.com", nil, nil)
		_, err := suite.repo.Create(suite.ctx, "alicex", "alice4@test.com", nil, nil)

		suite.Require().ErrorIs(err, core.ErrDuplicate)
	})
	suite.Cleanup()
}

func (suite *userRepositoryTestSuite) TestGet() {
	t := suite.T()

	t.Run("should get existing user", func(t *testing.T) {
		id := suite.fixtures.InsertUser(fixtures.RandomUserRow())

		u, err := suite.repo.Get(suite.ctx, id)

		suite.Require().NoError(err)
		suite.Require().Equal(id, u.ID)
	})

	t.Run("should return not found for invalid id", func(t *testing.T) {
		_, err := suite.repo.Get(suite.ctx, "invalid")

		suite.Require().ErrorIs(err, core.ErrNotFound)
	})
	suite.Cleanup()
}

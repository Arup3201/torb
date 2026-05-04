package manual

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/Arup3201/torb/models"
	"github.com/Arup3201/torb/testdata"
	"github.com/Arup3201/torb/testhelpers"
	"github.com/Arup3201/torb/testhelpers/fixtures"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var USER_ONE, USER_TWO string

type manualRepositoryTestSuite struct {
	suite.Suite
	pgContainer *testhelpers.PostgresContainer
	db          *gorm.DB
	fixtures    *fixtures.Fixtures
	repo        *ManualAccountRepository
	ctx         context.Context
}

func (suite *manualRepositoryTestSuite) SetupSuite() {
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

	suite.repo = NewManualAccountRepository(suite.db)

	err = testdata.TestMigrate(suite.db)
	if err != nil {
		log.Fatal(err)
	}

	// ensure ManualAccount table exists for these tests
	if err := suite.db.AutoMigrate(&models.ManualAccount{}); err != nil {
		log.Fatal(err)
	}

	suite.fixtures = fixtures.New(suite.ctx, suite.db)

	USER_ONE = suite.fixtures.InsertUser(fixtures.RandomUserRow())
	USER_TWO = suite.fixtures.InsertUser(fixtures.RandomUserRow())
}

func (suite *manualRepositoryTestSuite) Cleanup() {
	err := suite.db.WithContext(suite.ctx).
		Exec("DELETE FROM manual_accounts").Error
	suite.Require().NoError(err)
}

func TestManualRepository(t *testing.T) {
	suite.Run(t, new(manualRepositoryTestSuite))
}

func (suite *manualRepositoryTestSuite) TestCreate() {
	t := suite.T()

	t.Run("should create manual account", func(t *testing.T) {
		password := []byte("secret")

		err := suite.repo.Create(suite.ctx, USER_ONE, "user@test.com", password, false)

		suite.Cleanup()

		suite.Require().NoError(err)
	})
	t.Run("should give error with duplicate user id", func(t *testing.T) {
		password := "secret"
		suite.fixtures.InsertManualAccount(fixtures.GetManualAccount(USER_ONE, "user1@test.com", password))

		err := suite.repo.Create(suite.ctx, USER_ONE, "user2@test.com", []byte(password), false)

		suite.Cleanup()

		suite.Require().Error(err)
	})
	t.Run("should give error with duplicate email", func(t *testing.T) {
		password := "secret"
		suite.fixtures.InsertManualAccount(fixtures.GetManualAccount(USER_ONE, "user@test.com", password))

		err := suite.repo.Create(suite.ctx, USER_TWO, "user@test.com", []byte(password), false)

		suite.Cleanup()

		suite.Require().Error(err)
	})
}

func (suite *manualRepositoryTestSuite) TestGet() {
	t := suite.T()

	t.Run("should get existing account by user id", func(t *testing.T) {
		email := USER_ONE + "-get@example.com"
		acc := fixtures.GetManualAccount(USER_ONE, email, "pw")
		suite.fixtures.InsertManualAccount(acc)

		fetched, err := suite.repo.Get(suite.ctx, acc.UserID)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().Equal(acc.UserID, fetched.UserID)
		suite.Require().Equal(email, fetched.Email)
	})

	t.Run("should error when user id not found", func(t *testing.T) {
		_, err := suite.repo.Get(suite.ctx, "no-such-user")
		suite.Require().Error(err)
	})
}

func (suite *manualRepositoryTestSuite) TestGetByEmail() {
	t := suite.T()

	t.Run("should get by email", func(t *testing.T) {
		em := USER_ONE + "-email@example.com"
		acc := fixtures.GetManualAccount(USER_ONE, em, "pw")
		suite.fixtures.InsertManualAccount(acc)

		fetched, err := suite.repo.GetByEmail(suite.ctx, em)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().Equal(acc.UserID, fetched.UserID)
		suite.Require().Equal(em, fetched.Email)
	})
	t.Run("should get error with invalid email", func(t *testing.T) {
		_, err := suite.repo.GetByEmail(suite.ctx, "invalid email")

		suite.Cleanup()

		suite.Require().Error(err)
	})
}

func (suite *manualRepositoryTestSuite) TestUpdateVerificationToken() {
	t := suite.T()

	t.Run("should update verification token and fetch by token", func(t *testing.T) {
		acc := fixtures.GetManualAccount(USER_ONE, "user@test.com", "pw")
		suite.fixtures.InsertManualAccount(acc)

		token := "verif_tok_hash"
		expires := time.Now().Add(2 * time.Hour)
		err := suite.repo.UpdateVerificationToken(suite.ctx, acc, token, expires)

		suite.Cleanup()

		suite.Require().NoError(err)
	})
}

func (suite *manualRepositoryTestSuite) TestGetVerificationToken() {
	t := suite.T()

	t.Run("should get verification token", func(t *testing.T) {
		token := "verif_tok_hash"
		expires := time.Now().Add(2 * time.Hour)
		acc := fixtures.GetManualAccount(USER_ONE, "user@test.com", "pw")
		acc.VerificationToken = &token
		acc.VerificationTokenExpiresAt = &expires
		suite.fixtures.InsertManualAccount(acc)

		fetched, err := suite.repo.GetByVerificationToken(suite.ctx, token)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().NotNil(fetched.VerificationToken)
		suite.Require().Equal(token, *fetched.VerificationToken)
		suite.Require().NotNil(fetched.VerificationTokenExpiresAt)
	})
}

func (suite *manualRepositoryTestSuite) TestUpdateResetPasswordToken() {
	t := suite.T()

	t.Run("should update reset password token", func(t *testing.T) {
		token := "verif_tok_hash"
		expires := time.Now().Add(2 * time.Hour)
		acc := fixtures.GetManualAccount(USER_ONE, "user@test.com", "pw")
		acc.ResetPasswordToken = &token
		acc.ResetPasswordTokenExpiresAt = &[]time.Time{time.Now().UTC()}[0]
		suite.fixtures.InsertManualAccount(acc)

		err := suite.repo.UpdateResetPasswordToken(suite.ctx, acc, token, expires)

		suite.Cleanup()

		suite.Require().NoError(err)
	})
}

func (suite *manualRepositoryTestSuite) TestUpdateEmailVerified() {
	t := suite.T()

	t.Run("should update email_verified to true", func(t *testing.T) {
		acc := fixtures.GetManualAccount(USER_ONE, "user@test.com", "pw")

		err := suite.repo.UpdateEmailVerified(suite.ctx, acc, true)

		acc, _ = gorm.G[models.ManualAccount](suite.db).Where("user_id = ?", USER_ONE).First(suite.ctx)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().Equal(true, acc.EmailVerified)
	})
}

func (suite *manualRepositoryTestSuite) TestUpdatePassword() {
	t := suite.T()

	t.Run("should update password", func(t *testing.T) {
		acc := fixtures.GetManualAccount(USER_ONE, "user@test.com", "pw")
		password := []byte("newpw")

		err := suite.repo.UpdatePassword(suite.ctx, acc, password)

		acc, _ = gorm.G[models.ManualAccount](suite.db).Where("user_id = ?", USER_ONE).First(suite.ctx)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().Equal(password, acc.PasswordHash)
	})
}

package manual

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/Arup3201/torb/auth"
	"github.com/Arup3201/torb/core"
	"github.com/Arup3201/torb/models"
	"github.com/Arup3201/torb/testdata"
	"github.com/Arup3201/torb/testhelpers"
	"github.com/Arup3201/torb/testhelpers/fixtures"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var UID_PASSWORD string

type passwordServiceTestSuite struct {
	suite.Suite
	ctx         context.Context
	pgContainer *testhelpers.PostgresContainer
	db          *gorm.DB
	fixtures    *fixtures.Fixtures

	svc *PasswordService
}

func (suite *passwordServiceTestSuite) SetupSuite() {
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

	// ensure manual account table exists
	if err := suite.db.AutoMigrate(&models.ManualAccount{}); err != nil {
		log.Fatal(err)
	}

	repo := NewManualAccountRepository(suite.db)
	suite.svc = NewPasswordService(repo)

	suite.fixtures = fixtures.New(suite.ctx, suite.db)

	UID_PASSWORD = suite.fixtures.InsertUser(fixtures.RandomUserRow())
}

func (suite *passwordServiceTestSuite) Cleanup() {
	err := suite.db.WithContext(suite.ctx).
		Exec("DELETE FROM manual_accounts").Error
	suite.Require().NoError(err)
}

func TestPasswordService(t *testing.T) {
	suite.Run(t, new(passwordServiceTestSuite))
}

func (suite *passwordServiceTestSuite) TestGetResetToken() {
	t := suite.T()

	t.Run("should create reset token for existing email", func(t *testing.T) {
		uid := UID_PASSWORD
		email := "user" + uid + "@test.com"
		acc := fixtures.GetManualAccount(uid, email, "pw")
		suite.fixtures.InsertManualAccount(acc)

		token, err := suite.svc.GetResetToken(suite.ctx, email)
		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().NotEmpty(token)
	})

	t.Run("should error for unknown email", func(t *testing.T) {
		_, err := suite.svc.GetResetToken(suite.ctx, "no-such-email@example.com")

		suite.Cleanup()

		suite.Require().Error(err)
	})
}

func (suite *passwordServiceTestSuite) TestReset() {
	t := suite.T()

	t.Run("should reset password with valid token", func(t *testing.T) {
		uid := UID_PASSWORD
		email := "user" + uid + "@test.com"
		acc := fixtures.GetManualAccount(uid, email, "oldpw")
		// prepare reset token and expiry
		rawToken := "resettok123"
		sha := auth.GetTokenSHA(rawToken)
		exp := time.Now().UTC().Add(1 * time.Hour)
		acc.ResetPasswordToken = &sha
		acc.ResetPasswordTokenExpiresAt = &exp
		suite.fixtures.InsertManualAccount(acc)

		err := suite.svc.Reset(suite.ctx, rawToken, "newpassword")
		suite.Cleanup()

		suite.Require().NoError(err)
	})

	t.Run("should error for expired token", func(t *testing.T) {
		uid := UID_PASSWORD
		email := "user" + uid + "@test.com"
		acc := fixtures.GetManualAccount(uid, email, "oldpw")
		rawToken := "expiredtok"
		sha := auth.GetTokenSHA(rawToken)
		exp := time.Now().UTC().Add(-1 * time.Hour) // already expired
		acc.ResetPasswordToken = &sha
		acc.ResetPasswordTokenExpiresAt = &exp
		suite.fixtures.InsertManualAccount(acc)

		err := suite.svc.Reset(suite.ctx, rawToken, "newpassword")
		suite.Cleanup()

		suite.Require().Error(err)
		suite.Require().ErrorIs(err, core.ErrInvalidValue)
	})

	t.Run("should error for used token", func(t *testing.T) {
		uid := UID_PASSWORD
		email := "user" + uid + "@test.com"
		acc := fixtures.GetManualAccount(uid, email, "oldpw")
		rawToken := "expiredtok"
		sha := auth.GetTokenSHA(rawToken)
		exp := time.Now().UTC().Add(2 * time.Hour)
		acc.ResetPasswordToken = &sha
		acc.ResetPasswordTokenExpiresAt = &exp
		acc.ResetPasswordTokenUsedAt = &[]time.Time{time.Now().UTC()}[0]
		suite.fixtures.InsertManualAccount(acc)

		err := suite.svc.Reset(suite.ctx, rawToken, "newpassword")
		suite.Cleanup()

		suite.Require().Error(err)
		suite.Require().ErrorIs(err, core.ErrInvalidValue)
	})

	t.Run("should error for invalid token", func(t *testing.T) {
		uid := UID_PASSWORD
		email := "user" + uid + "@test.com"
		acc := fixtures.GetManualAccount(uid, email, "oldpw")
		// no token set
		suite.fixtures.InsertManualAccount(acc)

		err := suite.svc.Reset(suite.ctx, "randomtoken", "newpassword")
		suite.Cleanup()

		suite.Require().Error(err)
	})
}

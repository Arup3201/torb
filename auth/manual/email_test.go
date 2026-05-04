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

var UID_ONE string

type emailServiceTestSuite struct {
	suite.Suite
	ctx         context.Context
	pgContainer *testhelpers.PostgresContainer
	db          *gorm.DB
	fixtures    *fixtures.Fixtures

	repo *ManualAccountRepository
	svc  *EmailService
}

func (suite *emailServiceTestSuite) SetupSuite() {
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

	suite.repo = NewManualAccountRepository(suite.db)
	suite.svc = NewEmailService(suite.repo)

	suite.fixtures = fixtures.New(suite.ctx, suite.db)

	UID_ONE = suite.fixtures.InsertUser(fixtures.RandomUserRow())
}

func (suite *emailServiceTestSuite) Cleanup() {
	err := suite.db.WithContext(suite.ctx).
		Exec("DELETE FROM manual_accounts").Error
	suite.Require().NoError(err)
}

func TestEmailService(t *testing.T) {
	suite.Run(t, new(emailServiceTestSuite))
}

func (suite *emailServiceTestSuite) TestGetVerificationToken() {
	t := suite.T()

	t.Run("should create verification token for unverified account", func(t *testing.T) {
		uid := UID_ONE
		email := uid + "@example.com"
		acc := fixtures.GetManualAccount(uid, email, "pw")
		suite.fixtures.InsertManualAccount(acc)

		token, err := suite.svc.GetVerificationToken(suite.ctx, uid)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().NotEmpty(token)
	})

	t.Run("should error when account already verified", func(t *testing.T) {
		uid := UID_ONE
		em := uid + "@example.com"
		acc := fixtures.GetManualAccount(uid, em, "pw")
		acc.EmailVerified = true
		suite.fixtures.InsertManualAccount(acc)

		_, err := suite.svc.GetVerificationToken(suite.ctx, uid)

		suite.Cleanup()

		suite.Require().ErrorIs(err, core.ErrInvalidValue)
	})

	t.Run("should error when account not found", func(t *testing.T) {
		_, err := suite.svc.GetVerificationToken(suite.ctx, "no-such-user")
		suite.Require().Error(err)
	})
}

func (suite *emailServiceTestSuite) TestVerify() {
	t := suite.T()

	t.Run("should verify account with valid token", func(t *testing.T) {
		uid := UID_ONE
		em := uid + "@example.com"
		raw := "rawtok123"
		hash := auth.GetTokenSHA(raw)
		expires := time.Now().UTC().Add(1 * time.Hour)
		acc := models.ManualAccount{UserID: uid, Email: em, PasswordHash: []byte("pw"), VerificationToken: &hash, VerificationTokenExpiresAt: &expires}
		suite.fixtures.InsertManualAccount(acc)

		err := suite.svc.Verify(suite.ctx, raw)

		suite.Cleanup()

		suite.Require().NoError(err)
	})

	t.Run("should error on expired token", func(t *testing.T) {
		uid := UID_ONE
		em := uid + "@example.com"
		raw := "rawtok-exp"
		hash := auth.GetTokenSHA(raw)
		expires := time.Now().UTC().Add(-1 * time.Minute)

		acc := models.ManualAccount{UserID: uid, Email: em, PasswordHash: []byte("pw"), VerificationToken: &hash, VerificationTokenExpiresAt: &expires}
		suite.Require().NoError(suite.db.WithContext(suite.ctx).Create(&acc).Error)

		err := suite.svc.Verify(suite.ctx, raw)

		suite.Cleanup()

		suite.Require().ErrorIs(err, core.ErrInvalidValue)
	})

	t.Run("should error when token not found", func(t *testing.T) {
		err := suite.svc.Verify(suite.ctx, "unknown-token")
		suite.Require().Error(err)
	})
}

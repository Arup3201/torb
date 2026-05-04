package manual

import (
	"context"
	"log"
	"testing"

	"github.com/Arup3201/torb/core"
	"github.com/Arup3201/torb/core/users"
	"github.com/Arup3201/torb/models"
	"github.com/Arup3201/torb/testdata"
	"github.com/Arup3201/torb/testhelpers"
	"github.com/Arup3201/torb/testhelpers/fixtures"
	"github.com/stretchr/testify/suite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type registerServiceTestSuite struct {
	suite.Suite
	ctx         context.Context
	pgContainer *testhelpers.PostgresContainer
	db          *gorm.DB
	fixtures    *fixtures.Fixtures
	service     *RegisterService
}

func TestRegisterService(t *testing.T) {
	suite.Run(t, new(registerServiceTestSuite))
}

func (suite *registerServiceTestSuite) SetupSuite() {
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

	// ensure ManualAccount table exists for these tests
	if err := suite.db.AutoMigrate(&models.ManualAccount{}); err != nil {
		log.Fatal(err)
	}

	txManager := core.NewTxManager(suite.db)
	accountRepo := NewManualAccountRepository(suite.db)
	userRepo := users.NewUserRepository(suite.db)
	suite.service = NewRegisterService(txManager, accountRepo, userRepo)

	suite.fixtures = fixtures.New(suite.ctx, suite.db)

}

func (suite *registerServiceTestSuite) Cleanup() {
	err := suite.db.WithContext(suite.ctx).
		Exec("DELETE FROM users").Error
	suite.Require().NoError(err)
}

func (suite *registerServiceTestSuite) TestRegister() {
	t := suite.T()

	t.Run("should register user", func(t *testing.T) {
		username := "test"
		email := "test@example.com"
		password := "pw"
		dn := "Test"

		_, err := suite.service.CreateAccount(suite.ctx, username, email, password, &dn)

		suite.Cleanup()

		suite.Require().NoError(err)
	})
	t.Run("should save correct user details", func(t *testing.T) {
		username := "test"
		email := "test@example.com"
		password := "pw"
		dn := "Test"

		id, _ := suite.service.CreateAccount(suite.ctx, username, email, password, &dn)

		user, err := gorm.G[models.User](suite.db).Where("id = ?", id).First(suite.ctx)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().Equal(username, user.Username)
		suite.Require().Equal(email, user.Email)
		suite.Require().Equal(dn, *user.DisplayName)
	})
	t.Run("should save correct manual account details", func(t *testing.T) {
		username := "test"
		email := "test@example.com"
		password := "pw"
		dn := "Test"

		id, _ := suite.service.CreateAccount(suite.ctx, username, email, password, &dn)

		account, err := gorm.G[models.ManualAccount](suite.db).Where("user_id = ?", id).First(suite.ctx)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().Equal(email, account.Email)
		suite.Require().Equal(false, account.EmailVerified)
		err = bcrypt.CompareHashAndPassword(
			account.PasswordHash,
			[]byte(password))
		suite.Require().NoError(err)
	})
	t.Run("should not register with invalid email", func(t *testing.T) {
		username := "test"
		email := "test@example"
		password := "pw"
		dn := "Test"

		_, err := suite.service.CreateAccount(suite.ctx, username, email, password, &dn)

		suite.Cleanup()

		suite.Require().ErrorIs(err, core.ErrInvalidValue)
	})
	t.Run("should not register with invalid username", func(t *testing.T) {
		username := ""
		email := "test@example.com"
		password := "pw"
		dn := "Test"

		_, err := suite.service.CreateAccount(suite.ctx, username, email, password, &dn)

		suite.Cleanup()

		suite.Require().ErrorIs(err, core.ErrInvalidValue)
	})
	t.Run("should not register with duplicate username", func(t *testing.T) {
		username := "test"
		email := "test@example.com"
		password := "pw"
		dn := "Test"
		suite.fixtures.InsertUser(models.User{
			ID:          fixtures.RandomUserRow().ID,
			Username:    username,
			Email:       email,
			DisplayName: &dn,
		})

		_, err := suite.service.CreateAccount(suite.ctx, username, email, password, &dn)

		suite.Cleanup()

		suite.Require().Error(err)
	})
	t.Run("should not register with empty password", func(t *testing.T) {
		username := "test"
		email := "test@example.com"
		password := ""
		dn := "Test"

		_, err := suite.service.CreateAccount(suite.ctx, username, email, password, &dn)

		suite.Cleanup()

		suite.Require().ErrorIs(err, core.ErrInvalidValue)
	})
}

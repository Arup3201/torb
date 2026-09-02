package users

import (
	"context"
	"log"
	"testing"

	"github.com/Arup3201/torb/core"
	"github.com/Arup3201/torb/core/documents"
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

type userServiceTestSuite struct {
	suite.Suite
	ctx         context.Context
	pgContainer *testhelpers.PostgresContainer
	db          *gorm.DB
	fixtures    *fixtures.Fixtures
	service     *UserService
}

func TestUserService(t *testing.T) {
	suite.Run(t, new(userServiceTestSuite))
}

func (suite *userServiceTestSuite) SetupSuite() {
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

	repo := NewUserRepository(suite.db)
	docRepo := documents.NewDocumentRepository(suite.db)
	cfg, err := config.LoadDefaultConfig(suite.ctx)
	suite.NoError(err)
	s3Client := s3.NewFromConfig(cfg)
	docStorage := documents.NewDocumentStorage(s3Client)
	suite.service = NewUserService(repo, docRepo, docStorage)

	suite.fixtures = fixtures.New(suite.ctx, suite.db)
}

func (suite *userServiceTestSuite) Cleanup() {
	err := suite.db.WithContext(suite.ctx).
		Exec("DELETE FROM users").Error
	suite.Require().NoError(err)
}

func (suite *userServiceTestSuite) TestGet() {
	t := suite.T()

	t.Run("should get existing user", func(t *testing.T) {
		username := "alice-service"
		email := "alice@svc.test"
		dn := "Alice"
		au := "users/02d4cef0-c789-4899-bbe0-bcc79743fe41/avatars/avatar.jpg"
		id := suite.fixtures.InsertUser(models.User{
			Username:    username,
			Email:       email,
			DisplayName: &dn,
			AvatarKey:   &au,
			Skills:      "C, Python",
		})

		u, err := suite.service.Get(suite.ctx, id)

		suite.Require().NoError(err)
		suite.Require().Equal(id, u.ID)
		suite.Require().Equal("alice-service", u.Username)
		suite.Require().Equal("alice@svc.test", u.Email)
		suite.Require().Equal("C, Python", u.Skills)
		suite.Require().NotNil(u.DisplayName)
		suite.Require().Equal(dn, *u.DisplayName)
		suite.Require().NotNil(u.AvatarURL)
		suite.Require().Contains(*u.AvatarURL, "https://torb.s3.ap-south-1.amazonaws.com/"+au)
	})

	t.Run("should return not found for invalid id", func(t *testing.T) {
		_, err := suite.service.Get(suite.ctx, "invalid-id")

		suite.Require().ErrorIs(err, core.ErrNotFound)
	})
	suite.Cleanup()
}

func (suite *userServiceTestSuite) TestProjectAndTaskCount() {
	t := suite.T()

	// 2 users, 3 projects, 4 tasks
	// user 1 created 1 project
	// user 2 created 2 projects
	// user 1 is a member of the 2 projects created by user 2
	// user 1 is assigned to 4 tasks part of project 1, 2 and 3
	// 3 tasks are completed and 1 task is ongoing
	u1 := suite.fixtures.InsertUser(fixtures.RandomUserRow())
	u2 := suite.fixtures.InsertUser(fixtures.RandomUserRow())
	p1 := suite.fixtures.InsertProject(fixtures.RandomProjectRow(u1))
	p2 := suite.fixtures.InsertProject(fixtures.RandomProjectRow(u2))
	p3 := suite.fixtures.InsertProject(fixtures.RandomProjectRow(u2))
	suite.fixtures.InsertMember(fixtures.GetMemberRow(p2, u1, core.ROLE_MEMBER))
	suite.fixtures.InsertMember(fixtures.GetMemberRow(p3, u1, core.ROLE_MEMBER))
	p1t1 := suite.fixtures.InsertTask(fixtures.RandomTaskRow(p1, core.TASK_STATUS_COMPLETED))
	p1t2 := suite.fixtures.InsertTask(fixtures.RandomTaskRow(p1, core.TASK_STATUS_COMPLETED))
	p2t1 := suite.fixtures.InsertTask(fixtures.RandomTaskRow(p2, core.TASK_STATUS_COMPLETED))
	p3t1 := suite.fixtures.InsertTask(fixtures.RandomTaskRow(p3, core.TASK_STATUS_ONGOING))
	suite.fixtures.InsertAssignee(fixtures.GetAssigneeRow(p1, p1t1, u1))
	suite.fixtures.InsertAssignee(fixtures.GetAssigneeRow(p1, p1t2, u1))
	suite.fixtures.InsertAssignee(fixtures.GetAssigneeRow(p2, p2t1, u1))
	suite.fixtures.InsertAssignee(fixtures.GetAssigneeRow(p3, p3t1, u1))

	t.Run("should get 3 projects and 3 tasks", func(t *testing.T) {
		res, err := suite.service.ProjectAndTaskCount(suite.ctx, u1)

		suite.Require().NoError(err)
		suite.Require().Equal(3, int(res.Projects))
		suite.Require().Equal(4, int(res.Tasks))
		suite.Require().Equal(3, int(res.CompletedTasks))
	})
}

func (suite *userServiceTestSuite) TestUpdateUser() {
	t := suite.T()

	userID := suite.fixtures.InsertUser(fixtures.RandomUserRow())

	t.Run("should update username, displayname, skills and timezone", func(t *testing.T) {
		update := UpdateUserBody{
			Username:    &[]string{"arupjana"}[0],
			DisplayName: &[]string{"Arup Jana"}[0],
			Skills:      &[]string{"C, Python"}[0],
			Timezone:    &[]string{"UTC+05:30"}[0],
		}

		err := suite.service.Update(suite.ctx, userID, update)

		suite.Require().NoError(err)

		user, err := gorm.G[models.User](suite.db).Where("id = ?", userID).First(suite.ctx)
		suite.Require().NoError(err)
		suite.Require().Equal("arupjana", user.Username)
		suite.Require().Equal("Arup Jana", *user.DisplayName)
		suite.Require().Equal("C, Python", user.Skills)
		suite.Require().Equal("UTC+05:30", user.Timezone)
	})
}

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

func (suite *userRepositoryTestSuite) TearDownSuite() {
	suite.Require().NoError(suite.pgContainer.Terminate(suite.ctx))
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
		id, err := suite.repo.Create(suite.ctx, "alice", "alice@test.com", "", nil, nil)

		suite.Require().NoError(err)
		suite.Require().NotEmpty(id)
	})

	t.Run("should create user with display name and avatar url", func(t *testing.T) {
		dn := "Alice"
		au := "http://avatar"
		id, err := suite.repo.Create(suite.ctx, "alice2", "alice2@test.com", "", &dn, &au)

		suite.Require().NoError(err)
		suite.Require().NotEmpty(id)

		u, err := gorm.G[models.User](suite.db).Where("id = ?", id).First(suite.ctx)
		suite.Require().NoError(err)
		suite.Require().Equal(dn, *u.DisplayName)
		suite.Require().Equal(au, *u.AvatarURL)
	})
	t.Run("should get duplicate value error with same username", func(t *testing.T) {
		suite.repo.Create(suite.ctx, "alice3", "alice3@test.com", "", nil, nil)
		_, err := suite.repo.Create(suite.ctx, "alice3", "alicex@test.com", "", nil, nil)

		suite.Require().ErrorIs(err, core.ErrDuplicate)
	})
	t.Run("should get duplicate value error with same email", func(t *testing.T) {
		suite.repo.Create(suite.ctx, "alice4", "alice4@test.com", "", nil, nil)
		_, err := suite.repo.Create(suite.ctx, "alicex", "alice4@test.com", "", nil, nil)

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

	t.Run("should get all user details", func(t *testing.T) {
		id := suite.fixtures.InsertUser(models.User{
			Username:    "test",
			Email:       "test@example.com",
			DisplayName: &[]string{"Test"}[0],
			Skills:      "C, Python",
			Timezone:    "UTC+05:30",
		})

		u, _ := suite.repo.Get(suite.ctx, id)

		suite.Require().Equal(id, u.ID)
		suite.Require().Equal("test", u.Username)
		suite.Require().Equal("test@example.com", u.Email)
		suite.Require().Equal("Test", *(u.DisplayName))
		suite.Require().Equal("C, Python", u.Skills)
		suite.Require().Equal("UTC+05:30", u.Timezone)
	})
	suite.Cleanup()
}

func (suite *userRepositoryTestSuite) TestCountProjectsTasksAndCompletedTasks() {
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

	t.Run("should get 3 projects", func(t *testing.T) {
		cnt, err := suite.repo.CountProjects(suite.ctx, u1)

		suite.Require().NoError(err)
		suite.Require().Equal(3, int(cnt))
	})
	t.Run("should get total 4 tasks", func(t *testing.T) {
		cnt, err := suite.repo.CountTasks(suite.ctx, u1)

		suite.Require().NoError(err)
		suite.Require().Equal(4, int(cnt))
	})
	t.Run("should get 3 completed tasks", func(t *testing.T) {
		cnt, err := suite.repo.CountCompletedTasks(suite.ctx, u1)

		suite.Require().NoError(err)
		suite.Require().Equal(3, int(cnt))
	})
	suite.Cleanup()
}

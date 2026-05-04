package projects

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

type projectRepositoryTestSuite struct {
	suite.Suite
	pgContainer *testhelpers.PostgresContainer
	db          *gorm.DB
	fixtures    *fixtures.Fixtures
	repo        *ProjectRepository
	ctx         context.Context
}

func (suite *projectRepositoryTestSuite) SetupSuite() {
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

	suite.repo = NewProjectRepository(suite.db)

	err = testdata.TestMigrate(suite.db)
	if err != nil {
		log.Fatal(err)
	}

	suite.fixtures = fixtures.New(suite.ctx, suite.db)

	USER_ONE = suite.fixtures.InsertUser(fixtures.RandomUserRow())
	USER_TWO = suite.fixtures.InsertUser(fixtures.RandomUserRow())
	USER_THREE = suite.fixtures.InsertUser(fixtures.RandomUserRow())
}

func (suite *projectRepositoryTestSuite) Cleanup() {
	err := suite.db.WithContext(suite.ctx).
		Exec("DELETE FROM projects").Error
	suite.Require().NoError(err)
}

func TestProjectRepository(t *testing.T) {
	suite.Run(t, new(projectRepositoryTestSuite))
}

func (suite *projectRepositoryTestSuite) TestProjectCreate() {
	t := suite.T()

	t.Run("should create project with title description and skills", func(t *testing.T) {
		sample_name := "Test Project"
		sample_description := "Test Description"
		sample_skills := "C++, Java"

		_, err := suite.repo.Create(suite.ctx,
			sample_name, &sample_description, &sample_skills,
			USER_ONE)

		suite.Cleanup()

		suite.Require().NoError(err)
	})
	t.Run("should create project with only title", func(t *testing.T) {
		sample_name := "Test Project"

		_, err := suite.repo.Create(suite.ctx,
			sample_name, nil, nil,
			USER_ONE)

		suite.Cleanup()

		suite.Require().NoError(err)
	})
	t.Run("should create project with title and description", func(t *testing.T) {
		sample_name := "Test Project"
		sample_description := "Test Description"
		sample_skills := "C++, Java"

		_, err := suite.repo.Create(suite.ctx,
			sample_name, &sample_description, &sample_skills,
			USER_ONE)

		suite.Cleanup()

		suite.Require().NoError(err)
	})
	t.Run("should not create project with invalid user", func(t *testing.T) {
		sample_name := "Test Project"
		sample_description := "Test Description"
		sample_skills := "C++, Java"

		_, err := suite.repo.Create(suite.ctx,
			sample_name, &sample_description, &sample_skills,
			USER_ONE+"1234")

		suite.Cleanup()

		suite.Require().Error(err)
	})
}

func (suite *projectRepositoryTestSuite) TestProjectGet() {
	t := suite.T()

	t.Run("should get project summary with title description and skills", func(t *testing.T) {
		name, description, skills := "test project", "test description", "c, python"
		projectID := suite.fixtures.InsertProject(models.Project{
			Name:        name,
			Description: &description,
			Skills:      &skills,
			OwnerID:     USER_ONE,
		})

		project, err := suite.repo.Get(suite.ctx, projectID)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().Equal(projectID, project.ID)
		suite.Require().Equal(name, project.Name)
		suite.Require().Equal(description, *project.Description)
		suite.Require().Equal(skills, *project.Skills)
		suite.Require().Equal(USER_ONE, project.OwnerID)
	})
	t.Run("should get project summary with 1 ongoing task", func(t *testing.T) {
		projectID := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertTask(fixtures.RandomTaskRow(projectID, core.TASK_STATUS_ONGOING))

		project, _ := suite.repo.Get(suite.ctx, projectID)

		suite.Cleanup()

		suite.Require().EqualValues(1, project.OngoingTasks)
	})
	t.Run("should get project summary with 2 completed and 1 unassigned task", func(t *testing.T) {
		projectID := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertTask(fixtures.RandomTaskRow(projectID, core.TASK_STATUS_ONGOING))
		suite.fixtures.InsertTask(fixtures.RandomTaskRow(projectID, core.TASK_STATUS_COMPLETED))
		suite.fixtures.InsertTask(fixtures.RandomTaskRow(projectID, core.TASK_STATUS_COMPLETED))

		project, _ := suite.repo.Get(suite.ctx, projectID)

		suite.Cleanup()

		suite.Require().EqualValues(1, project.OngoingTasks)
		suite.Require().EqualValues(2, project.CompletedTasks)
	})
	t.Run("should not get any project with invalid id", func(t *testing.T) {
		projectID := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))

		_, err := suite.repo.Get(suite.ctx, projectID+"123")

		suite.Cleanup()

		suite.Require().ErrorIs(err, core.ErrNotFound)
	})
}

func (suite *projectRepositoryTestSuite) TestProjectList() {
	t := suite.T()

	t.Run("should return empty list", func(t *testing.T) {
		projects, err := suite.repo.List(suite.ctx, USER_ONE)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().NotNil(projects)
		suite.Require().Equal(0, len(projects))
	})
	t.Run("should return 2 projects", func(t *testing.T) {
		suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))

		projects, err := suite.repo.List(suite.ctx, USER_ONE)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().Equal(2, len(projects))
	})
}

func (suite *projectRepositoryTestSuite) TestProjectPublic() {
	t := suite.T()

	t.Run("should get empty list of public projects", func(t *testing.T) {
		projects, err := suite.repo.Public(suite.ctx, USER_ONE)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().NotNil(projects)
		suite.Require().Equal(0, len(projects))
	})
	t.Run("should get 2 public projects", func(t *testing.T) {
		suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))

		projects, err := suite.repo.Public(suite.ctx, USER_TWO)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().Equal(2, len(projects))
	})
	t.Run("should get 1 public project", func(t *testing.T) {
		suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		// should not be included in the result
		suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_TWO))

		projects, err := suite.repo.Public(suite.ctx, USER_TWO)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().Equal(1, len(projects))
	})
}

func (suite *projectRepositoryTestSuite) TestProjectRecentlyCreated() {
	t := suite.T()

	t.Run("should get no recently created projects", func(t *testing.T) {

		projects, err := suite.repo.RecentlyCreated(suite.ctx, USER_ONE, 10)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().NotNil(projects)
		suite.Require().Equal(0, len(projects))
	})
	t.Run("should get 2 recently created projects", func(t *testing.T) {
		suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))

		projects, err := suite.repo.RecentlyCreated(suite.ctx, USER_ONE, 10)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().Equal(2, len(projects))
	})
}

func (suite *projectRepositoryTestSuite) TestProjectRecentlyJoined() {
	t := suite.T()

	t.Run("should get no recently joined projects", func(t *testing.T) {

		projects, err := suite.repo.RecentlyJoined(suite.ctx, USER_TWO, 10)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().NotNil(projects)
		suite.Require().Equal(0, len(projects))
	})
	t.Run("should get 1 recently joined projects", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertMember(fixtures.GetMemberRow(p, USER_TWO, core.ROLE_MEMBER))

		projects, err := suite.repo.RecentlyJoined(suite.ctx, USER_TWO, 10)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().Equal(1, len(projects))
	})
	t.Run("should get 2 recently joined projects", func(t *testing.T) {
		p1 := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		p2 := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertMember(fixtures.GetMemberRow(p1, USER_TWO, core.ROLE_MEMBER))
		suite.fixtures.InsertMember(fixtures.GetMemberRow(p2, USER_TWO, core.ROLE_MEMBER))

		projects, err := suite.repo.RecentlyJoined(suite.ctx, USER_TWO, 10)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().Equal(2, len(projects))
	})
}

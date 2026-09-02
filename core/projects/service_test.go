package projects

import (
	"context"
	"log"
	"testing"

	"github.com/Arup3201/torb/core"
	"github.com/Arup3201/torb/core/members"
	"github.com/Arup3201/torb/models"
	"github.com/Arup3201/torb/testdata"
	"github.com/Arup3201/torb/testhelpers"
	"github.com/Arup3201/torb/testhelpers/fixtures"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type projectServiceTestSuite struct {
	suite.Suite
	ctx         context.Context
	pgContainer *testhelpers.PostgresContainer
	db          *gorm.DB
	fixtures    *fixtures.Fixtures
	service     *ProjectService
}

func TestProjectService(t *testing.T) {
	suite.Run(t, new(projectServiceTestSuite))
}

func (suite *projectServiceTestSuite) SetupSuite() {
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
	projectRepo := NewProjectRepository(suite.db)
	memberRepo := members.NewMemberRepository(suite.db)
	suite.service = NewProjectService(txManager, projectRepo, memberRepo)

	suite.fixtures = fixtures.New(suite.ctx, suite.db)

	USER_ONE = suite.fixtures.InsertUser(fixtures.RandomUserRow())
	USER_TWO = suite.fixtures.InsertUser(fixtures.RandomUserRow())
	USER_THREE = suite.fixtures.InsertUser(fixtures.RandomUserRow())
}

func (suite *projectServiceTestSuite) Cleanup() {
	err := suite.db.WithContext(suite.ctx).
		Exec("DELETE FROM projects").Error
	suite.Require().NoError(err)
}

func (suite *projectServiceTestSuite) TestProjectCreate() {
	t := suite.T()

	t.Run("should create project without error", func(t *testing.T) {
		sample_name := "Test Project"
		sample_description := "Test Description"
		sample_skills := "C++, Java"

		_, err := suite.service.Create(suite.ctx,
			sample_name,
			&sample_description,
			&sample_skills,
			USER_ONE,
		)

		suite.Cleanup()

		suite.Require().NoError(err)
	})
	t.Run("should create project with title description skills and not nil timestamps", func(t *testing.T) {
		sample_name := "Test Project"
		sample_description := "Test Description"
		sample_skills := "C++, Java"

		id, _ := suite.service.Create(suite.ctx,
			sample_name,
			&sample_description,
			&sample_skills,
			USER_ONE,
		)

		p, _ := gorm.G[models.Project](suite.db).
			Where("id = ?", id).First(suite.ctx)
		suite.Cleanup()

		suite.Require().Equal(sample_name, p.Name)
		suite.Require().Equal(sample_description, *p.Description)
		suite.Require().Equal(sample_skills, *p.Skills)
		suite.Require().Equal(USER_ONE, p.OwnerID)
		suite.Require().NotNil(p.CreatedAt)
		suite.Require().NotNil(p.UpdatedAt)
	})
	t.Run("should create project without description", func(t *testing.T) {
		sample_name := "Test Project"
		sample_skills := "C++, Java"

		_, err := suite.service.Create(suite.ctx,
			sample_name,
			nil,
			&sample_skills,
			USER_ONE,
		)

		suite.Cleanup()

		suite.Require().NoError(err)
	})
	t.Run("should creating project without skills", func(t *testing.T) {
		sample_name := "Test Project"
		sample_description := "Test Description"

		_, err := suite.service.Create(suite.ctx,
			sample_name,
			&sample_description,
			nil,
			USER_ONE,
		)

		suite.Cleanup()

		suite.Require().NoError(err)
	})
	t.Run("should fail to create project with empty name", func(t *testing.T) {
		sample_name := ""
		sample_description := "Test Description"
		sample_skills := "C++, Java"

		_, err := suite.service.Create(suite.ctx,
			sample_name,
			&sample_description,
			&sample_skills,
			USER_ONE,
		)

		suite.Cleanup()

		suite.Require().Error(err)
	})
	t.Run("should make project creator owner of the project", func(t *testing.T) {
		sample_name := "Test Project"
		sample_description := "Test Description"
		sample_skills := "C++, Java"

		id, _ := suite.service.Create(suite.ctx,
			sample_name,
			&sample_description,
			&sample_skills,
			USER_ONE,
		)

		m, err := gorm.G[models.Member](suite.db).
			Where("project_id = ? AND user_id = ?", id, USER_ONE).First(suite.ctx)
		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().Equal("Owner", m.Role.String)
	})
}

func (suite *projectServiceTestSuite) TestProjectGet() {
	t := suite.T()

	t.Run("should get project summary with name description skills", func(t *testing.T) {
		sample_name := "Test Project"
		sample_description := "Test Description"
		sample_skills := "C++, Java"
		projectID := suite.fixtures.InsertProject(models.Project{
			Name:        sample_name,
			Description: &sample_description,
			Skills:      &sample_skills,
			OwnerID:     USER_ONE,
		})

		res, err := suite.service.Get(suite.ctx, projectID)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().Equal(sample_name, res.Name)
		suite.Require().Equal(sample_description, *res.Description)
		suite.Require().Equal(sample_skills, *res.Skills)
	})
	t.Run("should get 1 ongoing and 2 completed task in project", func(t *testing.T) {
		projectID := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertTask(fixtures.RandomTaskRow(projectID, core.TASK_STATUS_ONGOING))
		suite.fixtures.InsertTask(fixtures.RandomTaskRow(projectID, core.TASK_STATUS_COMPLETED))
		suite.fixtures.InsertTask(fixtures.RandomTaskRow(projectID, core.TASK_STATUS_COMPLETED))

		project, _ := suite.service.Get(suite.ctx, projectID)

		suite.Cleanup()

		suite.Require().EqualValues(1, project.OngoingTasks)
		suite.Require().EqualValues(2, project.CompletedTasks)
	})
}

func (suite *projectServiceTestSuite) TestProjectUpdate() {
	t := suite.T()

	t.Run("should update project details for the owner", func(t *testing.T) {

		projectID := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))

		newName := "Updated Project"
		newDescription := "Updated description"
		newSkills := "Go, PostgreSQL"

		err := suite.service.Update(suite.ctx, projectID, USER_ONE,
			&newName, &newDescription, &newSkills)

		suite.Require().NoError(err)

		updated, err := gorm.G[models.Project](suite.db).
			Where("id = ?", projectID).First(suite.ctx)
		suite.Require().NoError(err)
		suite.Cleanup()
		suite.Require().Equal(newName, updated.Name)
		suite.Require().Equal(newDescription, *updated.Description)
		suite.Require().Equal(newSkills, *updated.Skills)
	})

	t.Run("should fail to update when user is not the owner", func(t *testing.T) {
		projectID := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))

		newName := "Updated Project"
		err := suite.service.Update(suite.ctx, projectID, USER_TWO, &newName, nil, nil)

		suite.Cleanup()

		suite.Require().Error(err)
	})
}

func (suite *projectServiceTestSuite) TestProjectMyProjects() {
	t := suite.T()

	t.Run("should get empty my projects list", func(t *testing.T) {
		projects, err := suite.service.MyProjects(suite.ctx, USER_ONE)

		suite.Require().NoError(err)
		suite.Require().NotNil(projects)
		suite.Require().Equal(0, len(projects))
	})
	t.Run("should get 1 project in my projects list", func(t *testing.T) {
		suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))

		projects, err := suite.service.MyProjects(suite.ctx, USER_ONE)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().Equal(1, len(projects))
	})
	t.Run("should have correct project summary in the first item", func(t *testing.T) {
		sample_name := "Test Project"
		sample_description := "Test Description"
		sample_skills := "C++, Java"
		projectID := suite.fixtures.InsertProject(models.Project{
			Name:        sample_name,
			Description: &sample_description,
			Skills:      &sample_skills,
			OwnerID:     USER_ONE,
		})
		suite.fixtures.InsertTask(fixtures.RandomTaskRow(projectID, core.TASK_STATUS_ONGOING))
		suite.fixtures.InsertTask(fixtures.RandomTaskRow(projectID, core.TASK_STATUS_COMPLETED))
		suite.fixtures.InsertTask(fixtures.RandomTaskRow(projectID, core.TASK_STATUS_COMPLETED))

		projects, _ := suite.service.MyProjects(suite.ctx, USER_ONE)

		suite.Cleanup()

		suite.Require().Equal(projectID, projects[0].ID)
		suite.Require().Equal(sample_name, projects[0].Name)
		suite.Require().Equal(sample_description, *projects[0].Description)
		suite.Require().Equal(sample_skills, *projects[0].Skills)
		suite.Require().EqualValues(1, projects[0].OngoingTasks)
		suite.Require().EqualValues(2, projects[0].CompletedTasks)
	})
}

func (suite *projectServiceTestSuite) TestProjectRecentlyCreated() {
	t := suite.T()

	t.Run("should get no recently created projects", func(t *testing.T) {

		projects, err := suite.service.RecentlyCreated(suite.ctx, USER_ONE)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().NotNil(projects)
		suite.Require().Equal(0, len(projects))
	})
	t.Run("should get 2 recently created projects", func(t *testing.T) {
		suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))

		projects, err := suite.service.RecentlyCreated(suite.ctx, USER_ONE)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().Equal(2, len(projects))
	})
	t.Run("should get correct project summary in the first item", func(t *testing.T) {
		sample_name := "Test Project"
		sample_description := "Test Description"
		sample_skills := "C++, Java"
		projectID := suite.fixtures.InsertProject(models.Project{
			Name:        sample_name,
			Description: &sample_description,
			Skills:      &sample_skills,
			OwnerID:     USER_ONE,
		})
		suite.fixtures.InsertTask(fixtures.RandomTaskRow(projectID, core.TASK_STATUS_ONGOING))
		suite.fixtures.InsertTask(fixtures.RandomTaskRow(projectID, core.TASK_STATUS_COMPLETED))
		suite.fixtures.InsertTask(fixtures.RandomTaskRow(projectID, core.TASK_STATUS_COMPLETED))

		projects, _ := suite.service.RecentlyCreated(suite.ctx, USER_ONE)

		suite.Cleanup()

		suite.Require().Equal(projectID, projects[0].ID)
		suite.Require().Equal(sample_name, projects[0].Name)
		suite.Require().Equal(sample_description, *projects[0].Description)
		suite.Require().Equal(sample_skills, *projects[0].Skills)
		suite.Require().EqualValues(1, projects[0].OngoingTasks)
		suite.Require().EqualValues(2, projects[0].CompletedTasks)
	})
}

func (suite *projectServiceTestSuite) TestProjectRecentlyJoined() {
	t := suite.T()

	t.Run("should get no recently joined projects", func(t *testing.T) {
		projects, err := suite.service.RecentlyJoined(suite.ctx, USER_TWO)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().NotNil(projects)
		suite.Require().Equal(0, len(projects))
	})
	t.Run("should get 2 recently joined projects", func(t *testing.T) {
		p1 := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		p2 := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertMember(fixtures.GetMemberRow(p1, USER_TWO, core.ROLE_MEMBER))
		suite.fixtures.InsertMember(fixtures.GetMemberRow(p2, USER_TWO, core.ROLE_MEMBER))

		projects, err := suite.service.RecentlyJoined(suite.ctx, USER_TWO)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().Equal(2, len(projects))
	})
	t.Run("should get correct project summary in the first item", func(t *testing.T) {
		sample_name := "Test Project"
		sample_description := "Test Description"
		sample_skills := "C++, Java"
		projectID := suite.fixtures.InsertProject(models.Project{
			Name:        sample_name,
			Description: &sample_description,
			Skills:      &sample_skills,
			OwnerID:     USER_ONE,
		})
		suite.fixtures.InsertTask(fixtures.RandomTaskRow(projectID, core.TASK_STATUS_ONGOING))
		suite.fixtures.InsertTask(fixtures.RandomTaskRow(projectID, core.TASK_STATUS_COMPLETED))
		suite.fixtures.InsertTask(fixtures.RandomTaskRow(projectID, core.TASK_STATUS_COMPLETED))
		suite.fixtures.InsertMember(fixtures.GetMemberRow(projectID, USER_TWO, core.ROLE_MEMBER))

		projects, _ := suite.service.RecentlyJoined(suite.ctx, USER_TWO)

		suite.Cleanup()

		suite.Require().Equal(projectID, projects[0].ID)
		suite.Require().Equal(sample_name, projects[0].Name)
		suite.Require().Equal(sample_description, *projects[0].Description)
		suite.Require().Equal(sample_skills, *projects[0].Skills)
		suite.Require().EqualValues(1, projects[0].OngoingTasks)
		suite.Require().EqualValues(2, projects[0].CompletedTasks)
	})
}

package members

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

type memberRepositoryTestSuite struct {
	suite.Suite
	pgContainer *testhelpers.PostgresContainer
	db          *gorm.DB
	fixtures    *fixtures.Fixtures
	repo        *MemberRepository
	ctx         context.Context
}

func (suite *memberRepositoryTestSuite) SetupSuite() {
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

	suite.repo = NewMemberRepository(suite.db)

	err = testdata.TestMigrate(suite.db)
	if err != nil {
		log.Fatal(err)
	}

	suite.fixtures = fixtures.New(suite.ctx, suite.db)

	USER_ONE = suite.fixtures.InsertUser(fixtures.RandomUserRow())
	USER_TWO = suite.fixtures.InsertUser(fixtures.RandomUserRow())
	USER_THREE = suite.fixtures.InsertUser(fixtures.RandomUserRow())
}

func (suite *memberRepositoryTestSuite) Cleanup() {
	err := suite.db.WithContext(suite.ctx).
		Exec("DELETE FROM projects").Error
	suite.Require().NoError(err)
}

func TestMemberRepository(t *testing.T) {
	suite.Run(t, new(memberRepositoryTestSuite))
}

func (suite *memberRepositoryTestSuite) TestCreate() {
	t := suite.T()

	t.Run("should create member", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))

		err := suite.repo.Create(suite.ctx, p, USER_TWO, core.ROLE_MEMBER)

		suite.Cleanup()

		suite.Require().NoError(err)
	})

	t.Run("should create member with Member role", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))

		suite.repo.Create(suite.ctx, p, USER_TWO, core.ROLE_MEMBER)

		var role string
		gorm.G[models.Member](suite.db).Select("role").Where("project_id = ? AND user_id = ?", p, USER_TWO).Scan(suite.ctx, &role)

		suite.Cleanup()

		suite.Require().Equal(core.ROLE_MEMBER, role)
	})

	t.Run("should not create member with invalid role", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))

		err := suite.repo.Create(suite.ctx, p, USER_TWO, "UNKNOWN")

		suite.Cleanup()

		suite.Require().Error(err)
	})
}

func (suite *memberRepositoryTestSuite) TestRole() {
	t := suite.T()

	t.Run("should get Owner role", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))

		role, err := suite.repo.Role(suite.ctx, p, USER_ONE)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().Equal(core.ROLE_OWNER, role)
	})

	t.Run("should get Member role", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertMember(fixtures.GetMemberRow(p, USER_TWO, core.ROLE_MEMBER))

		role, err := suite.repo.Role(suite.ctx, p, USER_TWO)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().Equal(core.ROLE_MEMBER, role)
	})

	t.Run("should get not found error", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))

		_, err := suite.repo.Role(suite.ctx, p, USER_TWO)

		suite.Cleanup()

		suite.Require().ErrorIs(err, core.ErrNotFound)
	})
}

func (suite *memberRepositoryTestSuite) TestCount() {
	t := suite.T()

	t.Run("should return 1 for owner", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))

		cnt, err := suite.repo.Count(suite.ctx, p)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().EqualValues(1, cnt)
	})

	t.Run("should return 2 after adding a member", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertMember(fixtures.GetMemberRow(p, USER_TWO, core.ROLE_MEMBER))

		cnt, err := suite.repo.Count(suite.ctx, p)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().EqualValues(2, cnt)
	})
}

func (suite *memberRepositoryTestSuite) TestList() {
	t := suite.T()

	t.Run("should give 1 member (owner)", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))

		rows, err := suite.repo.List(suite.ctx, p)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().NotNil(rows)
		suite.Require().Equal(1, len(rows))
	})

	t.Run("should give 2 members", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertMember(fixtures.GetMemberRow(p, USER_TWO, core.ROLE_MEMBER))

		rows, err := suite.repo.List(suite.ctx, p)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().Equal(2, len(rows))
	})

	t.Run("should give 2 members with correct values", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertMember(fixtures.GetMemberRow(p, USER_TWO, core.ROLE_MEMBER))

		rows, _ := suite.repo.List(suite.ctx, p)

		suite.Cleanup()

		suite.Require().ElementsMatch(
			[]string{USER_ONE, USER_TWO},
			[]string{rows[0].UserID, rows[1].UserID},
		)
	})
}

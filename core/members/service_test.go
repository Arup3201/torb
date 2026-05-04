package members

import (
	"context"
	"log"
	"testing"

	"github.com/Arup3201/torb/core"
	"github.com/Arup3201/torb/testdata"
	"github.com/Arup3201/torb/testhelpers"
	"github.com/Arup3201/torb/testhelpers/fixtures"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type memberServiceTestSuite struct {
	suite.Suite
	ctx         context.Context
	pgContainer *testhelpers.PostgresContainer
	db          *gorm.DB
	fixtures    *fixtures.Fixtures
	service     *MemberService
}

func TestMemberService(t *testing.T) {
	suite.Run(t, new(memberServiceTestSuite))
}

func (suite *memberServiceTestSuite) SetupSuite() {
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

	memberRepo := NewMemberRepository(suite.db)
	service := NewMemberService(memberRepo)
	suite.service = service

	suite.fixtures = fixtures.New(suite.ctx, suite.db)

	USER_ONE = suite.fixtures.InsertUser(fixtures.RandomUserRow())
	USER_TWO = suite.fixtures.InsertUser(fixtures.RandomUserRow())
	USER_THREE = suite.fixtures.InsertUser(fixtures.RandomUserRow())
}

func (suite *memberServiceTestSuite) Cleanup() {
	err := suite.db.WithContext(suite.ctx).
		Exec("DELETE FROM projects").Error
	suite.Require().NoError(err)
}

func (suite *memberServiceTestSuite) TestGetRole() {
	t := suite.T()

	t.Run("should return owner role", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))

		role, err := suite.service.GetRole(suite.ctx, p, USER_ONE)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().Equal(core.ROLE_OWNER, role)
	})

	t.Run("should return member role", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertMember(fixtures.GetMemberRow(p, USER_TWO, core.ROLE_MEMBER))

		role, err := suite.service.GetRole(suite.ctx, p, USER_TWO)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().Equal(core.ROLE_MEMBER, role)
	})
}

func (suite *memberServiceTestSuite) TestCount() {
	t := suite.T()

	t.Run("should return 1 for owner", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))

		cnt, err := suite.service.Count(suite.ctx, p, USER_ONE)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().EqualValues(1, cnt)
	})

	t.Run("should return 2 after adding a member", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertMember(fixtures.GetMemberRow(p, USER_TWO, core.ROLE_MEMBER))

		cnt, err := suite.service.Count(suite.ctx, p, USER_ONE)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().EqualValues(2, cnt)
	})
}

func (suite *memberServiceTestSuite) TestAllMembers() {
	t := suite.T()

	t.Run("should return all members when requester is a member", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertMember(fixtures.GetMemberRow(p, USER_TWO, core.ROLE_MEMBER))

		members, err := suite.service.AllMembers(suite.ctx, p, USER_ONE)

		suite.Cleanup()

		suite.Require().NoError(err)
		suite.Require().Equal(2, len(members))
	})

	t.Run("should return forbidden when requester is not a member", func(t *testing.T) {
		p := suite.fixtures.InsertProject(fixtures.RandomProjectRow(USER_ONE))
		suite.fixtures.InsertMember(fixtures.GetMemberRow(p, USER_TWO, core.ROLE_MEMBER))

		_, err := suite.service.AllMembers(suite.ctx, p, USER_THREE)

		suite.Cleanup()

		suite.Require().ErrorIs(err, core.ErrForbidden)
	})
}

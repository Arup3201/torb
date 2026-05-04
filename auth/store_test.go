package auth

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/Arup3201/torb/testhelpers"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"
)

type tokenStoreTestSuite struct {
	suite.Suite
	redis *redis.Client
	store *TokenStore
	ctx   context.Context
}

func (suite *tokenStoreTestSuite) SetupSuite() {
	var err error

	suite.ctx = context.Background()

	redisContainer, err := testhelpers.CreateRedisContainer(suite.ctx)
	if err != nil {
		log.Fatal(err)
	}

	connString, err := redisContainer.ConnectionString(suite.ctx)
	if err != nil {
		log.Fatal(err)
	}

	opt, err := redis.ParseURL(connString)
	if err != nil {
		log.Fatal(err)
	}

	suite.redis = redis.NewClient(opt)

	suite.store = NewTokenStore(suite.redis)
}

func (suite *tokenStoreTestSuite) Delete(key string) {
	suite.Require().NoError(suite.redis.Del(suite.ctx, key).Err())
}

func TestTokenStore(t *testing.T) {
	suite.Run(t, new(tokenStoreTestSuite))
}

func (suite *tokenStoreTestSuite) TestSave() {
	t := suite.T()

	t.Run("should save token", func(t *testing.T) {
		err := suite.store.Save(suite.ctx, "123", time.Now().Add(1*time.Minute))
		suite.Require().NoError(err)
		suite.Delete(getTokenKey("123"))
	})
	t.Run("should save token with revoked as false", func(t *testing.T) {
		suite.store.Save(suite.ctx, "123", time.Now().Add(1*time.Minute))

		revoked, err := suite.redis.HGet(suite.ctx, getTokenKey("123"), "revoked").Bool()
		suite.Require().NoError(err)
		suite.Require().False(revoked)
		suite.Delete(getTokenKey("123"))
	})
}

func (suite *tokenStoreTestSuite) TestRevoke() {
	t := suite.T()

	t.Run("should revoke token", func(t *testing.T) {
		suite.redis.HSet(suite.ctx, getTokenKey("123"), "revoked", false)

		err := suite.store.Revoke(suite.ctx, "123")

		suite.Require().NoError(err)

		revoked, err := suite.redis.HGet(suite.ctx, getTokenKey("123"), "revoked").Bool()
		suite.Require().NoError(err)
		suite.Require().True(revoked)

		suite.Delete(getTokenKey("123"))
	})
}

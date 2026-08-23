package api

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Arup3201/torb/auth"
	"github.com/Arup3201/torb/core"
	"github.com/Arup3201/torb/core/members"
	"github.com/Arup3201/torb/core/projects"
	"github.com/Arup3201/torb/core/requests"
	"github.com/Arup3201/torb/core/tasks"
	"github.com/Arup3201/torb/core/users"
	"github.com/Arup3201/torb/middlewares"
	"github.com/Arup3201/torb/models"
	"github.com/Arup3201/torb/notifications"
	"github.com/Arup3201/torb/testdata"
	"github.com/Arup3201/torb/testhelpers"
	"github.com/Arup3201/torb/testhelpers/fixtures"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var TEST_USER, TEST_PROJECT string

func TestRealtimeNotification(t *testing.T) {
	suite.Run(t, new(realtimeTestSuite))
}

type realtimeTestSuite struct {
	suite.Suite
	tokenService *auth.TokenService
	muxHandler   *http.ServeMux
	wsServer     *httptest.Server
	url          string
}

func (suite *realtimeTestSuite) SetupSuite() {
	ctx := context.Background()

	pgContainer, err := testhelpers.CreatePostgresContainer(ctx)
	if err != nil {
		log.Fatal(err)
	}

	db, err := gorm.Open(postgres.Open(pgContainer.ConnectionString), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	err = testdata.TestMigrate(db)
	if err != nil {
		log.Fatal(err)
	}

	db.AutoMigrate(&models.Notification{})

	f := fixtures.New(ctx, db)
	TEST_USER = f.InsertUser(fixtures.RandomUserRow())
	TEST_PROJECT = f.InsertProject(fixtures.RandomProjectRow(TEST_USER))

	redisContainer, err := testhelpers.CreateRedisContainer(ctx)
	if err != nil {
		log.Fatal(err)
	}

	connString, err := redisContainer.ConnectionString(ctx)
	if err != nil {
		log.Fatal(err)
	}

	opt, err := redis.ParseURL(connString)
	if err != nil {
		log.Fatal(err)
	}

	redis := redis.NewClient(opt)
	store := auth.NewTokenStore(redis)
	block, _ := pem.Decode([]byte(`-----BEGIN PRIVATE KEY-----
MIICdQIBADANBgkqhkiG9w0BAQEFAASCAl8wggJbAgEAAoGBANCFXFZd4xUB7KpZ
UYAre4pgwwSbiukMfGZ9f1tnFdTghzFxlVWXkGHwe3h8qsq9fdSVv/ES6REYBLDg
hmyPseO3BWBcGEAhoBo9YJlnXF8sZYcDzRTTME+lUSHd85Cipqw4hrA4JIKzMihg
QqhYKtuHCi71roq7UnReavzSfIDPAgMBAAECgYBOQs9OJvy0hL1jjhRVq3w5immH
UC2JnEMQYGetUXpTJFX5S60Fq9XnvE9LAFdFsmsIn4+jljpdTQttq0codaIIrIkG
RtbeX+7c4VxUdtR8Jv9XdW+l6olQnbolY5/SXG6MpYLQ8GCuRz8Wr6Cl3lef0uu/
59DjmoMtI3oVvTJGQQJBAPVVTR6g4NX2d9F9Hp2fPullv60TB2dP8liSknUhfr2i
7s0rdDed9O0cZ9aUwGFRGvxIYOEp6TR2ify/3rSw6mECQQDZllFgPLY5LQuBo36V
0J62DUsqe3ItWBgJHfzkJod1ScCP11pRYIxWl9RDdGCPKclCEC1mKKcx4cIIwKN/
3hkvAkA/NTQCYSasWazzL05VA/NchNeGivGMX5+rzE+pl/CkgTcPa1OtBKhW8sua
EIckS5YtS6SSPo8T8jqJARIq8a3hAkA7zRB4frcmZ7bt3l2AF2JHbsfl2R+8TqXs
e41xtxUrqyV9YxaznvFzKy9viqCvODDUM1YG6c1p7D5D4Y4OKqCJAkAFk7/ZDQrr
1X4DLmAFDkjxwoasHOGAn13xeh4dtbF9iW1WPljlKGlhIVaqiJN5JyNafWxBPNL2
VRE5pSFPQliu
-----END PRIVATE KEY-----
`))
	parseResult, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		log.Fatal(err)
	}

	privateKey := parseResult.(*rsa.PrivateKey)
	suite.tokenService = auth.NewTokenService(store, "test", privateKey)

	txManager := core.NewTxManager(db)
	projectRepo := projects.NewProjectRepository(db)
	memberRepo := members.NewMemberRepository(db)
	userRepo := users.NewUserRepository(db)
	joinRepo := requests.NewJoinRepository(db)
	notificationRepo := notifications.NewNotificationRepository(db)
	taskRepo := tasks.NewTaskRepository(db)
	projectService := projects.NewProjectService(
		txManager,
		projectRepo,
		memberRepo)
	userService := users.NewUserService(userRepo, nil, nil)
	memberService := members.NewMemberService(memberRepo)
	joinService := requests.NewJoinRequestService(
		txManager,
		joinRepo,
		memberRepo,
	)
	notificationService := notifications.NewNotificationService(
		projectRepo,
		taskRepo,
		memberRepo,
		userRepo,
		notificationRepo,
	)

	wsHandler := NewWebSocketConnectionHandler(suite.tokenService)
	projectHandler := NewProjectApi(
		projectService,
		userService,
		memberService,
		joinService,
		notificationService)
	suite.muxHandler = http.NewServeMux()
	suite.muxHandler.Handle("GET /", http.HandlerFunc(wsHandler.WebSocketConnector))
	authenticator := middlewares.NewAuthenticator(suite.tokenService)
	suite.muxHandler.Handle("POST /projects/{id}/join-requests", middlewares.HTTPErrorHandler(authenticator.IsAuthenticated(projectHandler.AddJoinRequest)))
	suite.wsServer = httptest.NewServer(http.HandlerFunc(wsHandler.WebSocketConnector))
	suite.url = strings.Replace(suite.wsServer.URL, "http", "ws", 1)
}

func (suite *realtimeTestSuite) Cleanup() {
	suite.wsServer.Close()
}

func (suite *realtimeTestSuite) TestWebSocketHandshake() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	c, _, err := websocket.Dial(ctx, suite.url, nil)
	suite.Require().NoError(err)
	defer c.CloseNow()

	c.Close(websocket.StatusNormalClosure, "")
}

func (suite *realtimeTestSuite) TestWebSocketSend2Messages() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	c, _, err := websocket.Dial(ctx, suite.url, nil)
	suite.Require().NoError(err)
	defer c.CloseNow()

	err = wsjson.Write(ctx, c, map[string]string{
		"type":  "token",
		"token": "random",
	})
	suite.Require().NoError(err)

	err = wsjson.Write(ctx, c, map[string]string{
		"type":  "token",
		"token": "random",
	})
	suite.Require().NoError(err)

	c.Close(websocket.StatusNormalClosure, "")
}

func (suite *realtimeTestSuite) TestWebSocketAuthentication() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	c, _, err := websocket.Dial(ctx, suite.url, nil)
	suite.Require().NoError(err)
	defer c.CloseNow()

	accessToken, err := suite.tokenService.CreateAccessToken(ctx, TEST_USER)
	suite.Require().NoError(err)
	err = wsjson.Write(ctx, c, map[string]string{
		"type":  "token",
		"token": accessToken.Value,
	})
	suite.Require().NoError(err)

	var v map[string]string
	err = wsjson.Read(ctx, c, &v)
	suite.Require().NoError(err)
	suite.Require().Equal("ack", v["type"])

	c.Close(websocket.StatusNormalClosure, "")
}

func (suite *realtimeTestSuite) TestWebSocketAuthenticationFail() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	c, _, err := websocket.Dial(ctx, suite.url, nil)
	suite.Require().NoError(err)
	defer c.CloseNow()

	err = wsjson.Write(ctx, c, map[string]string{
		"type":  "token",
		"token": "random",
	})
	suite.Require().NoError(err)

	var v map[string]string
	err = wsjson.Read(ctx, c, &v)
	suite.Require().NoError(err)
	suite.Require().Equal("error", v["type"])

	c.Close(websocket.StatusNormalClosure, "")
}

func (suite *realtimeTestSuite) TestWebSocketJoinRequestNotification() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	c, _, err := websocket.Dial(ctx, suite.url, nil)
	suite.Require().NoError(err)
	defer c.CloseNow()

	accessToken, err := suite.tokenService.CreateAccessToken(ctx, TEST_USER)
	suite.Require().NoError(err)
	err = wsjson.Write(ctx, c, map[string]string{
		"type":  "token",
		"token": accessToken.Value,
	})
	suite.Require().NoError(err)
	var v map[string]string
	err = wsjson.Read(ctx, c, &v)
	suite.Require().NoError(err)

	req := httptest.NewRequest("POST", "/projects/"+TEST_PROJECT+"/join-requests", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken.Value)
	res := httptest.NewRecorder()

	suite.muxHandler.ServeHTTP(res, req)
	suite.Require().NoError(err)
	suite.Require().Equal(http.StatusOK, res.Result().StatusCode)

	var msg struct {
		Type string                       `json:"type"`
		Data *notifications.JoinRequested `json:"data"`
	}
	err = wsjson.Read(ctx, c, &msg)
	suite.Require().NoError(err)
	suite.Require().Equal(notifications.NT_JOIN_REQUESTED, msg.Type)

	c.Close(websocket.StatusNormalClosure, "")
}

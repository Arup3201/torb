package app

import (
	"crypto/rsa"
	"fmt"
	"net/http"
	"time"

	"github.com/Arup3201/torb/api"
	"github.com/Arup3201/torb/auth"
	"github.com/Arup3201/torb/auth/manual"
	"github.com/Arup3201/torb/auth/openid"
	"github.com/Arup3201/torb/core"
	"github.com/Arup3201/torb/core/assignees"
	"github.com/Arup3201/torb/core/comments"
	"github.com/Arup3201/torb/core/documents"
	"github.com/Arup3201/torb/core/members"
	"github.com/Arup3201/torb/core/projects"
	"github.com/Arup3201/torb/core/requests"
	"github.com/Arup3201/torb/core/tasks"
	"github.com/Arup3201/torb/core/users"
	"github.com/Arup3201/torb/middlewares"
	"github.com/Arup3201/torb/notifications"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/redis/go-redis/v9"
	"github.com/resend/resend-go/v3"
	"github.com/rs/cors"
	"gorm.io/gorm"
)

type patternWithHandler struct {
	method, pattern string
	handler         middlewares.HTTPErrorHandler
}

type App struct {
	// CORS allowed origins
	AllowedCrossOrigins []string

	config  *Config
	handler *http.ServeMux
}

func NewApp(
	// API paths start with Prefix: /v1, /v2, /api, /v1/api
	prefix string,

	config *Config,
	db *gorm.DB,
	redis *redis.Client,
	privateKey *rsa.PrivateKey,
	s3Client *s3.Client,
	frontendLoginUrl string,
	frontendVerifyUrl string,
	frontendResetUrl string,
	frontendHomeUrl string,
) *App {
	handler := http.NewServeMux()

	memberRepo := members.NewMemberRepository(db)
	joinRepo := requests.NewJoinRepository(db)
	accountRepo := manual.NewManualAccountRepository(db)
	oauthRepo := openid.NewOauthRepository(db)
	userRepo := users.NewUserRepository(db)
	docRepo := documents.NewDocumentRepository(db)
	assigneeRepo := assignees.NewAssigneeRepository(db)
	commentRepo := comments.NewCommentRepository(db)
	projectRepo := projects.NewProjectRepository(db)
	taskRepo := tasks.NewTaskRepository(db)
	notificationRepo := notifications.NewNotificationRepository(db)
	txManager := core.NewTxManager(db)
	tokenStore := auth.NewTokenStore(redis)
	cache := core.NewCacheManager(redis)
	stringStore := openid.NewStringStore(redis)
	docStorage := documents.NewDocumentStorage(s3Client)

	memberService := members.NewMemberService(memberRepo, docStorage)
	joinService := requests.NewJoinRequestService(
		txManager,
		joinRepo,
		memberRepo,
		docStorage,
	)
	userService := users.NewUserService(userRepo, docRepo, docStorage)
	assigneeService := assignees.NewAssigneeService(
		memberRepo,
		assigneeRepo)
	commentService := comments.NewCommentService(
		commentRepo,
		memberRepo,
		docStorage)
	projectService := projects.NewProjectService(
		txManager,
		projectRepo,
		memberRepo)
	taskService := tasks.NewTaskService(
		taskRepo,
		memberRepo,
		assigneeRepo,
		docStorage,
	)
	notificationService := notifications.NewNotificationService(
		projectRepo,
		taskRepo,
		memberRepo,
		userRepo,
		notificationRepo,
	)
	registerService := manual.NewRegisterService(txManager, accountRepo, userRepo)
	tokenService := auth.NewTokenService(tokenStore, TOKEN_ISSUER, privateKey)
	emailService := manual.NewEmailService(accountRepo)
	passwordService := manual.NewPasswordService(accountRepo)
	googleService := openid.NewGoogleService(
		config.GoogleClientID,
		config.GoogleClientSecret,
		config.GoogleRedirectURI,
		config.GoogleConnectRedirectURI,
		txManager,
		userRepo,
		oauthRepo,
		stringStore,
	)

	authenticator := middlewares.NewAuthenticator(tokenService)

	resendClient := resend.NewClient(config.ResendApiKey)
	authApi := api.NewAuthApi(
		registerService,
		tokenService,
		emailService,
		passwordService,
		userService,
		resendClient,
		frontendVerifyUrl,
		frontendResetUrl,
	)
	googleApi := api.NewGoogleApi(
		googleService,
		tokenService,
		userService,
		frontendHomeUrl,
		frontendLoginUrl,
	)
	userApi := api.NewUserApi(userService, googleService, registerService)
	projectApi := api.NewProjectApi(
		projectService,
		userService,
		memberService,
		joinService,
		notificationService,
		cache,
		30*time.Minute,
	)
	taskApi := api.NewTaskApi(
		taskService,
		assigneeService,
		commentService,
		notificationService,
	)
	messageApi := api.NewMessageApi(notificationService)
	webSocketApi := api.NewWebSocketConnectionHandler(tokenService)

	patternWithHandlers := []patternWithHandler{
		// Auth
		{
			method:  "POST",
			pattern: "/auth/register",
			handler: authApi.Register,
		},
		{
			method:  "POST",
			pattern: "/auth/login",
			handler: authApi.Login,
		},
		{
			method:  "POST",
			pattern: "/auth/refresh",
			handler: authApi.Refresh,
		},
		{
			method:  "POST",
			pattern: "/auth/logout",
			handler: authApi.Logout,
		},
		{
			method:  "POST",
			pattern: "/auth/verify-email",
			handler: authApi.VerifyEmail,
		},
		{
			method:  "POST",
			pattern: "/auth/resend-verification",
			handler: authApi.ResendVerificationEmail,
		},
		{
			method:  "POST",
			pattern: "/auth/password-reset-email",
			handler: authApi.SendPasswordResetEmail,
		},
		{
			method:  "POST",
			pattern: "/auth/password-reset",
			handler: authApi.ResetPassword,
		},
		{
			method:  "GET",
			pattern: "/auth/google/redirect",
			handler: googleApi.Redirect,
		},
		{
			method:  "GET",
			pattern: "/auth/google/callback",
			handler: googleApi.Callback,
		},
		{
			method:  "POST",
			pattern: "/auth/google/login",
			handler: googleApi.Login,
		},

		{
			method:  "GET",
			pattern: "/profile",
			handler: authenticator.IsAuthenticated(userApi.GetProfileSummary),
		},
		{
			method:  "POST",
			pattern: "/profile/avatar",
			handler: authenticator.IsAuthenticated(userApi.UploadAvatar),
		},
		{
			method:  "POST",
			pattern: "/profile/add-password",
			handler: authenticator.IsAuthenticated(userApi.CreatePasswordAccount),
		},
		{
			method:  "GET",
			pattern: "/profile/google/redirect",
			handler: googleApi.Redirect2,
		},
		{
			method:  "GET",
			pattern: "/profile/google/callback",
			handler: googleApi.ConnectGoogleAccountCallback,
		},
		// List APIs
		{
			method:  "GET",
			pattern: "/projects",
			handler: authenticator.IsAuthenticated(projectApi.ListMyProjects),
		},
		{
			method:  "GET",
			pattern: "/dashboard/projects/created",
			handler: authenticator.IsAuthenticated(projectApi.ListRecentlyCreated),
		},
		{
			method:  "GET",
			pattern: "/dashboard/projects/joined",
			handler: authenticator.IsAuthenticated(projectApi.ListRecentlyJoined),
		},
		{
			method:  "GET",
			pattern: "/projects/{project_id}/tasks",
			handler: authenticator.IsAuthenticated(taskApi.List),
		},
		{
			method:  "GET",
			pattern: "/dashboard/tasks/assigned",
			handler: authenticator.IsAuthenticated(taskApi.ListAssignedTasks),
		},
		{
			method:  "GET",
			pattern: "/dashboard/tasks/unassigned",
			handler: authenticator.IsAuthenticated(taskApi.ListUnassignedTasks),
		},
		{
			method:  "GET",
			pattern: "/projects/{id}/members",
			handler: authenticator.IsAuthenticated(projectApi.ListMembers),
		},
		{
			method:  "GET",
			pattern: "/projects/{id}/join-requests",
			handler: authenticator.IsAuthenticated(projectApi.ListJoinRequests),
		},
		{
			method:  "GET",
			pattern: "/projects/{project_id}/tasks/{task_id}/comments",
			handler: authenticator.IsAuthenticated(taskApi.ListComments),
		},
		{
			method:  "GET",
			pattern: "/public/projects",
			handler: authenticator.IsAuthenticated(projectApi.ListPublic),
		},
		{
			method:  "GET",
			pattern: "/messages",
			handler: authenticator.IsAuthenticated(messageApi.List),
		},
		// Get Single Instance APIs
		{
			method:  "GET",
			pattern: "/projects/{id}",
			handler: authenticator.IsAuthenticated(projectApi.Get),
		},
		{
			method:  "GET",
			pattern: "/projects/{project_id}/tasks/{task_id}",
			handler: authenticator.IsAuthenticated(taskApi.Get),
		},
		// Create APIs
		{
			method:  "POST",
			pattern: "/projects",
			handler: authenticator.IsAuthenticated(projectApi.Create),
		},
		{
			method:  "POST",
			pattern: "/projects/{project_id}/tasks",
			handler: authenticator.IsAuthenticated(taskApi.Create),
		},
		{
			method:  "POST",
			pattern: "/projects/{id}/join-requests",
			handler: authenticator.IsAuthenticated(projectApi.AddJoinRequest),
		},
		{
			method:  "POST",
			pattern: "/projects/{project_id}/tasks/{task_id}/comments",
			handler: authenticator.IsAuthenticated(taskApi.AddComment),
		},
		// Update Instance APIs
		{
			method:  "PATCH",
			pattern: "/users",
			handler: authenticator.IsAuthenticated(userApi.UpdateProfile),
		},
		{
			method:  "PATCH",
			pattern: "/projects/{id}",
			handler: authenticator.IsAuthenticated(projectApi.Update),
		},
		{
			method:  "PATCH",
			pattern: "/projects/{id}/join-requests",
			handler: authenticator.IsAuthenticated(projectApi.RespondToJoinRequest),
		},
		{
			method:  "PATCH",
			pattern: "/projects/{project_id}/tasks/{task_id}",
			handler: authenticator.IsAuthenticated(taskApi.Update),
		},
		{
			method:  "PATCH",
			pattern: "/messages/{id}",
			handler: authenticator.IsAuthenticated(messageApi.MarkAsRead),
		},
	}

	for _, h := range patternWithHandlers {
		handler.Handle(
			h.method+" "+prefix+h.pattern,
			middlewares.HTTPErrorHandler(h.handler),
		)
	}

	// WebSocket Connection Accept/Upgrade API
	handler.HandleFunc("GET /", webSocketApi.WebSocketConnector)

	return &App{
		config:  config,
		handler: handler,
	}
}

func (app *App) Start() error {

	cors := cors.New(cors.Options{
		AllowedOrigins: app.AllowedCrossOrigins,
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowCredentials: true,
		AllowedHeaders:   []string{"*"},
	})

	// server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", app.config.Host, app.config.Port),
		Handler:      cors.Handler(app.handler),
		ReadTimeout:  20 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	fmt.Printf("[INFO] server starting at %s:%s\n", app.config.Host, app.config.Port)

	err := server.ListenAndServe()
	if err != nil {
		return fmt.Errorf("server listen and serve: %w", err)
	}

	return nil
}

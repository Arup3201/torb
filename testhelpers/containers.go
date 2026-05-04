package testhelpers

import (
	"context"
	"time"

	keycloak "github.com/stillya/testcontainers-keycloak"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
)

type PostgresContainer struct {
	*postgres.PostgresContainer
	ConnectionString string
}

func CreatePostgresContainer(ctx context.Context) (*PostgresContainer, error) {
	pgContainer, err := postgres.Run(ctx,
		"postgres:15.3-alpine",
		postgres.WithDatabase("test-db"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(5*time.Second)),
	)
	if err != nil {
		return nil, err
	}

	connString, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, err
	}

	return &PostgresContainer{
		PostgresContainer: pgContainer,
		ConnectionString:  connString,
	}, nil
}

func CreateKeycloakContainer(ctx context.Context) (*keycloak.KeycloakContainer, error) {
	keycloakContainer, err := keycloak.Run(ctx,
		"quay.io/keycloak/keycloak:26.4.7",
		testcontainers.WithWaitStrategy(wait.ForListeningPort("8080/tcp").WithStartupTimeout(2*time.Minute)),
		keycloak.WithContextPath("/auth"),
		keycloak.WithRealmImportFile("../testdata/ptracker-realm.json"),
		keycloak.WithAdminUsername("admin"),
		keycloak.WithAdminPassword("admin"),
	)
	return keycloakContainer, err
}

func CreateRedisContainer(ctx context.Context) (*redis.RedisContainer, error) {
	redisContainer, err := redis.Run(ctx, "redis:8.4.0-alpine")
	if err != nil {
		return nil, err
	}

	err = redisContainer.Start(ctx)
	if err != nil {
		return nil, err
	}

	return redisContainer, nil
}

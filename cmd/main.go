package main

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"

	"github.com/Arup3201/torb/cmd/app"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	API_ROOT             = "/api/v1"
	FRONTEND_LOGIN_PATH  = "/login"
	FRONTEND_VERIFY_PATH = "/verify-email"
	FRONTEND_RESET_PATH  = "/reset-password"
	FRONTEND_HOME_PATH   = "/"
)

func readRSAPrivateKey(filename string) (*rsa.PrivateKey, error) {
	bytes, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	block, _ := pem.Decode(bytes)
	parseResult, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("x509 parse pkce#1 private key: %w", err)
	}

	privateKey := parseResult.(*rsa.PrivateKey)
	return privateKey, nil
}

func main() {
	var err error
	ctx := context.Background()

	config := &app.Config{}
	err = config.LoadFromEnv()
	if err != nil {
		log.Fatalf("[ERROR] config load from env: %s", err)
	}

	db, err := gorm.Open(postgres.Open(fmt.Sprintf("host=%s port=%s user=%s "+
		"password=%s dbname=%s sslmode=disable", config.DBHost, config.DBPort,
		config.DBUser, config.DBPass, config.DBName)),
		&gorm.Config{})
	if err != nil {
		log.Fatalf("[ERROR] gorm open: %s", err)
	}

	app.Migrate(db)

	rdsOptions := &redis.Options{
		Addr:     fmt.Sprintf("%s:%s", config.RedisHost, config.RedisPort),
		Username: config.RedisUser,
		Password: config.RedisPass,
		DB:       0,
		Protocol: 2,
	}
	if config.RedisTLS == "true" {
		rdsOptions.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}
	redis := redis.NewClient(rdsOptions)
	if err := redis.Ping(ctx).Err(); err != nil {
		log.Fatalf("redis connection failed: %s", err)
	}

	privateKey, err := readRSAPrivateKey("private.key")
	if err != nil {
		log.Fatalf("rsa generate key: %s\n", err)
	}

	cfg, err := awsConfig.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatal(err)
	}
	s3Client := s3.NewFromConfig(cfg)

	app := app.NewApp(
		API_ROOT,
		config,
		db,
		redis,
		privateKey,
		s3Client,
		config.FrontendURL+FRONTEND_LOGIN_PATH,
		config.FrontendURL+FRONTEND_VERIFY_PATH,
		config.FrontendURL+FRONTEND_RESET_PATH,
		config.FrontendURL+FRONTEND_HOME_PATH,
	)
	app.AllowedCrossOrigins = []string{config.FrontendURL}
	err = app.Start()
	if err != nil {
		fmt.Printf("[ERROR] app start: %s", err)
	}
}

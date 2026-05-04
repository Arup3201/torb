package main

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"

	"github.com/Arup3201/torb/cmd/app"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	REDIS_ADDR           = "127.0.0.1:6379"
	API_ROOT             = "/api/v1"
	FRONTEND_URL         = "http://localhost:5173"
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

	redis := redis.NewClient(&redis.Options{
		Addr:     REDIS_ADDR,
		Password: "",
		DB:       0,
		Protocol: 2,
	})

	privateKey, err := readRSAPrivateKey("private.key")
	if err != nil {
		log.Fatalf("rsa generate key: %s\n", err)
	}

	app := app.NewApp(
		API_ROOT,
		config,
		db,
		redis,
		privateKey,
		FRONTEND_URL+FRONTEND_LOGIN_PATH,
		FRONTEND_URL+FRONTEND_VERIFY_PATH,
		FRONTEND_URL+FRONTEND_RESET_PATH,
		FRONTEND_URL+FRONTEND_HOME_PATH,
	)
	app.AllowedCrossOrigins = []string{FRONTEND_URL}
	err = app.Start()
	if err != nil {
		fmt.Printf("[ERROR] app start: %s", err)
	}
}

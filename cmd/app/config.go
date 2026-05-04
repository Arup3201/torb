package app

import (
	"fmt"
	"os"
)

type Config struct {
	Host, Port                                            string
	DBHost, DBPort, DBPass, DBUser, DBName                string
	ResendApiKey                                          string
	GoogleClientID, GoogleClientSecret, GoogleRedirectURI string
}

func (c *Config) LoadFromEnv() error {
	c.Host = os.Getenv(ENV_HOST)
	if c.Host == "" {
		return fmt.Errorf("environment variable '%s' missing", ENV_HOST)
	}
	c.Port = os.Getenv(ENV_PORT)
	if c.Port == "" {
		return fmt.Errorf("environment variable '%s' missing", ENV_PORT)
	}
	c.DBHost = os.Getenv(ENV_DB_HOST)
	if c.DBHost == "" {
		return fmt.Errorf("environment variable '%s' missing", ENV_DB_HOST)
	}
	c.DBUser = os.Getenv(ENV_DB_USER)
	if c.DBUser == "" {
		return fmt.Errorf("environment variable '%s' missing", ENV_DB_USER)
	}
	c.DBPort = os.Getenv(ENV_DB_PORT)
	if c.DBPort == "" {
		return fmt.Errorf("environment variable '%s' missing", ENV_DB_PORT)
	}
	c.DBPass = os.Getenv(ENV_DB_PASS)
	if c.DBPass == "" {
		return fmt.Errorf("environment variable '%s' missing", ENV_DB_PASS)
	}
	c.DBName = os.Getenv(ENV_DB_NAME)
	if c.DBName == "" {
		return fmt.Errorf("environment variable '%s' missing", ENV_DB_NAME)
	}
	c.ResendApiKey = os.Getenv(ENV_RESEND_API_KEY)
	if c.ResendApiKey == "" {
		return fmt.Errorf("environment variable '%s' missing", ENV_RESEND_API_KEY)
	}

	c.GoogleClientID = os.Getenv(ENV_GOOGLE_CLIENT_ID)
	if c.GoogleClientID == "" {
		return fmt.Errorf("environment variable '%s' missing", ENV_GOOGLE_CLIENT_ID)
	}
	c.GoogleClientSecret = os.Getenv(ENV_GOOGLE_CLIENT_SECRET)
	if c.GoogleClientSecret == "" {
		return fmt.Errorf("environment variable '%s' missing", ENV_GOOGLE_CLIENT_SECRET)
	}
	c.GoogleRedirectURI = os.Getenv(ENV_GOOGLE_REDIRECT_URI)
	if c.GoogleRedirectURI == "" {
		return fmt.Errorf("environment variable '%s' missing", ENV_GOOGLE_REDIRECT_URI)
	}

	return nil
}

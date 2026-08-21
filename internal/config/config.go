package config

import (
	"fmt"
	"os"
	"strconv"
)

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	To       string
}

type Config struct {
	DatabaseURL string
	ArchiveDir  string
	SMTP        SMTPConfig
}

func Load() (Config, error) {
	var cfg Config

	required := []struct {
		dst *string
		key string
	}{
		{&cfg.DatabaseURL, "DATABASE_URL"},
		{&cfg.SMTP.Host, "SMTP_HOST"},
		{&cfg.SMTP.Username, "SMTP_USERNAME"},
		{&cfg.SMTP.Password, "SMTP_PASSWORD"},
		{&cfg.SMTP.From, "SMTP_FROM"},
		{&cfg.SMTP.To, "SMTP_TO"},
	}

	for _, r := range required {
		val, err := requireEnv(r.key)
		if err != nil {
			return Config{}, err
		}
		*r.dst = val
	}

	cfg.ArchiveDir = os.Getenv("ARCHIVE_DIR")
	if cfg.ArchiveDir == "" {
		cfg.ArchiveDir = "./archive"
	}

	portStr, err := requireEnv("SMTP_PORT")
	if err != nil {
		return Config{}, err
	}
	cfg.SMTP.Port, err = strconv.Atoi(portStr)
	if err != nil {
		return Config{}, fmt.Errorf("invalid SMTP_PORT: %w", err)
	}

	return cfg, nil
}

func requireEnv(key string) (string, error) {
	value := os.Getenv(key)
	if len(value) == 0 {
		return "", fmt.Errorf("missing required env var: %s", key)
	}
	return value, nil
}

package config

import (
	"os"
	"testing"
)

func TestLoadConfigFromEnv(t *testing.T) {
	// 设置测试环境变量
	os.Setenv("SERVER_PORT", "9999")
	os.Setenv("SERVER_MODE", "release")
	os.Setenv("DATABASE_HOST", "k8s-postgres-svc")
	os.Setenv("DATABASE_PORT", "5433")
	os.Setenv("DATABASE_USER", "custom_user")
	os.Setenv("DATABASE_PASSWORD", "custom_password")
	os.Setenv("DATABASE_DBNAME", "custom_exam_db")
	os.Setenv("JWT_SECRET", "custom_jwt_secret_token_123")

	defer func() {
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("SERVER_MODE")
		os.Unsetenv("DATABASE_HOST")
		os.Unsetenv("DATABASE_PORT")
		os.Unsetenv("DATABASE_USER")
		os.Unsetenv("DATABASE_PASSWORD")
		os.Unsetenv("DATABASE_DBNAME")
		os.Unsetenv("JWT_SECRET")
	}()

	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Server.Port != 9999 {
		t.Errorf("expected Server.Port=9999, got %d", cfg.Server.Port)
	}
	if cfg.Server.Mode != "release" {
		t.Errorf("expected Server.Mode='release', got %s", cfg.Server.Mode)
	}
	if cfg.Database.Host != "k8s-postgres-svc" {
		t.Errorf("expected Database.Host='k8s-postgres-svc', got %s", cfg.Database.Host)
	}
	if cfg.Database.Port != 5433 {
		t.Errorf("expected Database.Port=5433, got %d", cfg.Database.Port)
	}
	if cfg.Database.User != "custom_user" {
		t.Errorf("expected Database.User='custom_user', got %s", cfg.Database.User)
	}
	if cfg.Database.Password != "custom_password" {
		t.Errorf("expected Database.Password='custom_password', got %s", cfg.Database.Password)
	}
	if cfg.Database.DBName != "custom_exam_db" {
		t.Errorf("expected Database.DBName='custom_exam_db', got %s", cfg.Database.DBName)
	}
	if cfg.JWT.Secret != "custom_jwt_secret_token_123" {
		t.Errorf("expected JWT.Secret='custom_jwt_secret_token_123', got %s", cfg.JWT.Secret)
	}
}

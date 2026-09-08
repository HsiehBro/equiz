package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	WeChat   WeChatConfig   `mapstructure:"wechat"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type DatabaseConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	DBName       string `mapstructure:"dbname"`
	SSLMode      string `mapstructure:"sslmode"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
}

func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=Asia/Shanghai",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode)
}

type JWTConfig struct {
	Secret      string `mapstructure:"secret"`
	ExpireHours int    `mapstructure:"expire_hours"`
}

type WeChatConfig struct {
	AppID     string `mapstructure:"app_id"`
	AppSecret string `mapstructure:"app_secret"`
}

var GlobalConfig *Config

func LoadConfig(path string) (*Config, error) {
	v := viper.New()

	// 1. 设置默认值（确保在无 YAML 文件时纯环境变量也能被完整 Unmarshal）
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.mode", "debug")
	v.SetDefault("database.host", "127.0.0.1")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.user", "postgres")
	v.SetDefault("database.password", "")
	v.SetDefault("database.dbname", "exam_prep")
	v.SetDefault("database.sslmode", "disable")
	v.SetDefault("database.max_idle_conns", 10)
	v.SetDefault("database.max_open_conns", 100)
	v.SetDefault("jwt.secret", "exam_prep_secret_key_2026_super_safe")
	v.SetDefault("jwt.expire_hours", 72)
	v.SetDefault("wechat.app_id", "")
	v.SetDefault("wechat.app_secret", "")

	if path == "" {
		path = os.Getenv("CONFIG_PATH")
	}

	if path != "" {
		v.SetConfigFile(path)
	} else {
		v.AddConfigPath("./configs")
		v.AddConfigPath("../configs")
		v.AddConfigPath("../../configs")
		v.SetConfigName("config")
		v.SetConfigType("yaml")
	}

	// 2. 允许环境变量覆盖配置 (例如 DATABASE_HOST 映射到 database.host)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		// 如果配置文件不存在，尝试依靠环境变量与默认值运行
		fmt.Fprintf(os.Stderr, "Info: config file not found or skipped, using environment variables and defaults: %v\n", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Server.Mode == "" {
		cfg.Server.Mode = "debug"
	}
	if cfg.JWT.Secret == "" {
		cfg.JWT.Secret = "exam_prep_default_jwt_secret_2026"
	}
	if cfg.JWT.ExpireHours == 0 {
		cfg.JWT.ExpireHours = 72
	}

	GlobalConfig = &cfg
	return &cfg, nil
}

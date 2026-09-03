package database

import (
	"fmt"
	"log"
	"time"

	"exam-server/internal/config"
	"exam-server/internal/model"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitDB(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	dsn := cfg.DSN()

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// 连接池参数设置
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 自动数据表迁移
	log.Println("[Database] Running AutoMigrate for PostgreSQL...")
	err = db.AutoMigrate(
		&model.User{},
		&model.QuestionBank{},
		&model.Question{},
		&model.UserRecord{},
		&model.UserError{},
		&model.UserFavorite{},
		&model.MockRecord{},
		&model.UserNote{},
		&model.UserNoteLike{},
		&model.StudyPlan{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to auto migrate database tables: %w", err)
	}

	// 自动修复历史遗留的 creator_id = 0 的私有题库（归属到默认开发测试用户 ID 1，确保可见且可删除）
	db.Model(&model.QuestionBank{}).Where("creator_id = 0 AND is_official = false AND visibility = 'private'").Update("creator_id", 1)

	DB = db
	log.Println("[Database] Connected and migrated PostgreSQL successfully")
	return db, nil
}

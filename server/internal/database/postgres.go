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
		&model.SystemNotification{},
		&model.UserReferral{},
		&model.VIPOrder{},
		&model.Category{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to auto migrate database tables: %w", err)
	}

	// 自动修复历史遗留的 creator_id = 0 的私有题库（归属到默认开发测试用户 ID 1，确保可见且可删除）
	db.Model(&model.QuestionBank{}).Where("creator_id = 0 AND is_official = false AND visibility = 'private'").Update("creator_id", 1)

	// 自动修复历史遗留的题目笔记/错题/作答记录中 bank_id 与关联试题真实 bank_id 不一致的问题
	db.Exec(`
		UPDATE user_notes 
		SET bank_id = questions.bank_id 
		FROM questions 
		WHERE user_notes.question_id = questions.id 
		  AND user_notes.bank_id != questions.bank_id
	`)
	db.Exec(`
		UPDATE user_errors 
		SET bank_id = questions.bank_id 
		FROM questions 
		WHERE user_errors.question_id = questions.id 
		  AND user_errors.bank_id != questions.bank_id
	`)
	db.Exec(`
		UPDATE user_records 
		SET bank_id = questions.bank_id 
		FROM questions 
		WHERE user_records.question_id = questions.id 
		  AND user_records.bank_id != questions.bank_id
	`)

	// 确保 system_notifications 表中存在 bank_title 字段并修正历史题目与题库不一致的旧快照
	db.Exec(`ALTER TABLE system_notifications ADD COLUMN IF NOT EXISTS bank_title varchar(255) DEFAULT '';`)
	db.Exec(`
		UPDATE system_notifications 
		SET question_title = questions.title,
		    bank_id = questions.bank_id
		FROM questions 
		WHERE system_notifications.question_id = questions.id 
		  AND system_notifications.question_id > 0
		  AND (system_notifications.question_title != questions.title OR system_notifications.question_title IS NULL);
	`)
	db.Exec(`
		UPDATE system_notifications
		SET bank_title = question_banks.title
		FROM question_banks
		WHERE system_notifications.bank_id = question_banks.id
		  AND system_notifications.bank_id > 0;
	`)

	// 确保 question_banks 表中存在 is_vip 与 category_id 字段
	db.Exec(`ALTER TABLE question_banks ADD COLUMN IF NOT EXISTS is_vip boolean DEFAULT false;`)
	db.Exec(`ALTER TABLE question_banks ADD COLUMN IF NOT EXISTS category_id integer DEFAULT 1;`)

	// 初始化内置基础分类（若 categories 为空）
	var catCount int64
	db.Model(&model.Category{}).Count(&catCount)
	if catCount == 0 {
		initialCategories := []model.Category{
			{
				ID:          1,
				Name:        "综合",
				Description: "综合性与通用测试题库",
				Icon:        "/assets/icons/grid_view_gray.svg",
				BgClass:     "bg-gray-light",
				SortOrder:   99,
				IsActive:    true,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			{
				ID:          2,
				Name:        "医学类",
				Description: "执业医师 / 药师 / 护理考证",
				Icon:        "/assets/icons/medical_services_primary.svg",
				BgClass:     "bg-blue-light",
				SortOrder:   1,
				IsActive:    true,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			{
				ID:          3,
				Name:        "财经类",
				Description: "CPA / 会计初级 / 金融从业",
				Icon:        "/assets/icons/account_balance_gray.svg",
				BgClass:     "bg-gray-light",
				SortOrder:   2,
				IsActive:    true,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			{
				ID:          4,
				Name:        "IT互联网",
				Description: "软考 / PMP / 程序员认证",
				Icon:        "/assets/icons/terminal_gray.svg",
				BgClass:     "bg-gray-light",
				SortOrder:   3,
				IsActive:    true,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
		}
		for _, cat := range initialCategories {
			db.Create(&cat)
		}
	}

	// 核心修复：同步 categories 表主键序列自增值，避免预设种子数据导致的 duplicate key 异常
	db.Exec(`SELECT setval(pg_get_serial_sequence('categories', 'id'), COALESCE((SELECT MAX(id) FROM categories), 1));`)

	// 自动修复历史遗留的空白或无效图标分类（如已添加的考研政治等自动回退为通用书籍图标）
	db.Exec(`UPDATE categories SET icon = '/assets/icons/menu_book_primary.svg', bg_class = 'bg-blue-light' WHERE icon = '' OR icon LIKE '%folder%' OR icon IS NULL;`)

	// 确保官方预设题库的分类与 VIP 标识同步，并更新 category_id
	db.Model(&model.QuestionBank{}).Where("title = ?", "项目管理基础考试").Updates(map[string]interface{}{
		"category":    "IT互联网",
		"category_id": 4,
		"is_vip":      true,
	})
	db.Model(&model.QuestionBank{}).Where("title = ?", "初级会计实务 - 核心考点").Updates(map[string]interface{}{
		"category":    "财经类",
		"category_id": 3,
		"is_vip":      true,
	})
	db.Model(&model.QuestionBank{}).Where("title = ?", "2023年护士执业资格考试").Updates(map[string]interface{}{
		"category":    "医学类",
		"category_id": 2,
		"is_vip":      false,
	})
	// 将其余 category_id 为 0 或 NULL 的题库默认归入综合分类 (ID 1)
	db.Exec(`UPDATE question_banks SET category_id = 1 WHERE category_id IS NULL OR category_id = 0;`)

	DB = db
	log.Println("[Database] Connected and migrated PostgreSQL successfully")
	return db, nil
}

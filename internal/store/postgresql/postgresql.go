package postgresql

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/shopspring/decimal"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	_ "github.com/jackc/pgx/v5/stdlib" // 注册 pgx driver 至 database/sql

	"Xin-api/internal/model"
)

// buildDSN 从独立字段拼接 PostgreSQL DSN
func buildDSN() string {
	host := os.Getenv("POSTGRES_HOST")
	if host == "" {
		host = "127.0.0.1"
	}
	user := os.Getenv("POSTGRES_USER")
	if user == "" {
		user = "root"
	}
	password := os.Getenv("POSTGRES_PASSWORD")
	dbname := os.Getenv("POSTGRES_DB")
	if dbname == "" {
		dbname = "gateway"
	}
	port := os.Getenv("POSTGRES_PORT")
	if port == "" {
		port = "5432"
	}
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Shanghai",
		host, user, password, dbname, port)
}

// parseDSN 从 PostgreSQL DSN 中提取数据库名，并生成连到 postgres 维护库的连接串
func parseDSN(dsn string) (dbName string, adminDSN string) {
	re := regexp.MustCompile(`\bdbname=(\S+)`)
	matches := re.FindStringSubmatch(dsn)
	if len(matches) < 2 {
		panic(fmt.Sprintf("无法从 POSTGRESQL_DSN 中解析 dbname: %s", dsn))
	}
	dbName = matches[1]
	adminDSN = re.ReplaceAllString(dsn, "dbname=postgres")
	return
}

// NewPostgres 初始化 PostgreSQL 连接，自动建库、建表、创建默认管理员账号
// adminInitPassword 为空时随机生成一次性口令，避免使用硬编码默认密码
func NewPostgres(usdCNYRate float64, adminInitPassword string) *gorm.DB {
	dsn := buildDSN()

	dbName, adminDSN := parseDSN(dsn)

	// 第一步：连接 postgres 维护库，确保目标数据库已存在
	adminDB, err := sql.Open("pgx", adminDSN)
	if err != nil {
		panic(fmt.Sprintf("连接 PostgreSQL 维护库失败: %v", err))
	}

	var exists bool
	if err := adminDB.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", dbName).Scan(&exists); err != nil {
		panic(fmt.Sprintf("检查数据库 %s 是否存在时失败: %v", dbName, err))
	}

	if !exists {
		if _, err := adminDB.Exec(fmt.Sprintf("CREATE DATABASE %q", dbName)); err != nil {
			// 并发场景下另一实例可能已创建，重检查
			var stillMissing bool
			adminDB.QueryRow("SELECT NOT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", dbName).Scan(&stillMissing)
			if stillMissing {
				panic(fmt.Sprintf("创建数据库 %s 失败: %v", dbName, err))
			}
			log.Printf("[DB] 数据库 %q 已由其他实例创建", dbName)
		} else {
			log.Printf("[DB] 数据库 %q 自动创建成功", dbName)
		}
	}
	adminDB.Close()

	// 第二步：通过 GORM 连接目标数据库
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Error,
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries:      true,
			Colorful:                  true,
		},
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		panic(fmt.Sprintf("连接 PostgreSQL 失败: %v", err))
	}
	log.Println("连接 PostgreSQL 成功")

	// AutoMigrate 不会修改生成列表达式：旧表缺少 generation_token_count 时，
	// 先删除旧口径的 tpot 生成列，再交由 AutoMigrate 按新表达式重建。
	var hasGenerationTokenCol bool
	if err := db.Raw(`SELECT EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_schema = current_schema()
		  AND table_name = 'channel_metrics'
		  AND column_name = 'generation_token_count'
	)`).Scan(&hasGenerationTokenCol).Error; err != nil {
		panic(fmt.Sprintf("检查 channel_metrics 表结构失败: %v", err))
	}
	if !hasGenerationTokenCol {
		if err := db.Exec(`ALTER TABLE IF EXISTS channel_metrics DROP COLUMN IF EXISTS tpot`).Error; err != nil {
			panic(fmt.Sprintf("重建 channel_metrics.tpot 生成列失败: %v", err))
		}
	}

	// 第三步：自动迁移（建表）
	// 顺序按外键依赖排列：先父表后子表
	if err := db.AutoMigrate(
		&model.User{},
		&model.Channel{},
		&model.ApiKey{},
		&model.Model{},
		&model.ModelPrice{},
		&model.ModelPriceBinding{},
		&model.RequestLog{},
		&model.RequestPayload{},
		&model.BillingRecord{},
		&model.ChannelMetric{},
		&model.AuditLog{},
		&model.ExchangeRate{},
	); err != nil {
		panic(fmt.Sprintf("数据库自动迁移失败: %v", err))
	}
	log.Println("[DB] AutoMigrate 完成")

	// 第四步：创建默认管理员账号（幂等）
	var adminCount int64
	db.Model(&model.User{}).Where("username = ?", "admin").Count(&adminCount)
	if adminCount == 0 {
		adminPassword := adminInitPassword
		if adminPassword == "" {
			adminPassword = generateRandomPassword()
			log.Printf("[DB] 未设置 ADMIN_INIT_PASSWORD，已生成随机管理员口令（仅本次显示，请登录后立即修改）: %s", adminPassword)
		}
		hashed, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
		if err != nil {
			panic(fmt.Sprintf("加密默认管理员密码失败: %v", err))
		}
		if err := db.Create(&model.User{
			Username:           "admin",
			Password:           string(hashed),
			Role:               model.UserRoleAdmin,    // 显式写入，满足 CHECK (role IN (...))
			Status:             model.UserStatusActive, // 显式写入，满足 CHECK (status IN (...))
			Currency:           "CNY",                  // 用户结算币种固定 CNY
			MustChangePassword: true,                   // 初始口令仅用于首次登录，登录后必须改密
		}).Error; err != nil {
			panic(fmt.Sprintf("创建默认管理员账号失败: %v", err))
		}
		log.Println("[DB] 默认管理员账号已创建: username=admin")
	}

	// 第五步：初始化 USD→CNY 汇率种子（已存在则不覆盖管理员修改）
	initialRate := decimal.NewFromFloat(usdCNYRate).Round(8)
	if initialRate.LessThanOrEqual(decimal.Zero) {
		initialRate = decimal.NewFromFloat(7.2)
	}
	if err := NewExchangeRateRepo(db).EnsureUSDCNYSeed(context.Background(), model.NewMoney(initialRate)); err != nil {
		panic(fmt.Sprintf("初始化 USD→CNY 汇率失败: %v", err))
	}

	// 第六步：存量非 CNY 用户余额按初始汇率一次性迁移至 CNY（幂等，迁移后条件不再命中）
	migrateResult := db.WithContext(context.Background()).Exec(`
UPDATE users SET
  balance = ROUND(balance * ?::numeric, 8),
  quota = CASE WHEN quota IS NULL THEN NULL ELSE ROUND(quota * ?::numeric, 8) END,
  used_quota = ROUND(used_quota * ?::numeric, 8),
  currency = 'CNY',
  updated_at = NOW()
WHERE currency IS DISTINCT FROM 'CNY'`, initialRate.String(), initialRate.String(), initialRate.String())
	if migrateResult.Error != nil {
		panic(fmt.Sprintf("迁移存量用户余额至 CNY 失败: %v", migrateResult.Error))
	}
	if migrateResult.RowsAffected > 0 {
		log.Printf("[DB] 已将 %d 个非 CNY 用户余额迁移为 CNY，汇率=%s", migrateResult.RowsAffected, initialRate.String())
	}

	return db
}

// generateRandomPassword 生成 32 位十六进制随机口令，用于首次初始化管理员账号
func generateRandomPassword() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		panic(fmt.Sprintf("生成随机管理员密码失败: %v", err))
	}
	return hex.EncodeToString(buf)
}

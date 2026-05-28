package postgresql

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"regexp"
	"time"

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
func NewPostgres() *gorm.DB {
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

	// 第三步：自动迁移（建表）
	if err := db.AutoMigrate(
		&model.User{},
		&model.Group{},
		&model.Channel{},
		&model.ApiKey{},
	); err != nil {
		panic(fmt.Sprintf("数据库自动迁移失败: %v", err))
	}
	log.Println("[DB] AutoMigrate 完成")

	// 第四步：创建默认管理员账号（幂等）
	var adminCount int64
	db.Model(&model.User{}).Where("username = ?", "admin").Count(&adminCount)
	if adminCount == 0 {
		hashed, err := bcrypt.GenerateFromPassword([]byte("12345678"), bcrypt.DefaultCost)
		if err != nil {
			panic(fmt.Sprintf("加密默认管理员密码失败: %v", err))
		}
		if err := db.Create(&model.User{
			Username: "admin",
			Password: string(hashed),
		}).Error; err != nil {
			panic(fmt.Sprintf("创建默认管理员账号失败: %v", err))
		}
		log.Println("[DB] 默认管理员账号已创建: username=admin password=12345678")
	}

	return db
}

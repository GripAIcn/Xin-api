package postgresql

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"Xin-api/internal/model"
)

// 测试用最小表结构：SQLite 不支持 PG 的 text[] / varchar[] 数组类型，
// 因此仅映射 ListEnabledModels 涉及到的列，避免直接 AutoMigrate 生产模型。
type testChannel struct {
	ID        int64          `gorm:"column:id;primaryKey;autoIncrement"`
	Name      string         `gorm:"column:name"`
	Status    string         `gorm:"column:status"`
	CreatedAt time.Time      `gorm:"column:created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (testChannel) TableName() string { return "channels" }

type testModel struct {
	ID        int64          `gorm:"column:id;primaryKey;autoIncrement"`
	ChannelID int64          `gorm:"column:channel_id"`
	Name      string         `gorm:"column:name"`
	Status    string         `gorm:"column:status"`
	CreatedAt time.Time      `gorm:"column:created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (testModel) TableName() string { return "models" }

// newChannelModelsTestDB 使用内存 SQLite 构建渠道/模型测试库
func newChannelModelsTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("获取底层连接失败: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	if err := db.AutoMigrate(&testChannel{}, &testModel{}); err != nil {
		t.Fatalf("迁移渠道/模型表失败: %v", err)
	}
	return db
}

// TestChannelRepoListEnabledModels 验证仅返回「模型 active + 渠道 active + 双方未软删」的逻辑模型，
// 且同名模型跨渠道去重，created_at 取最早时间。
func TestChannelRepoListEnabledModels(t *testing.T) {
	ctx := context.Background()
	db := newChannelModelsTestDB(t)
	repo := NewChannelRepo(db)

	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	newChannel := func(name, status string) *testChannel {
		ch := &testChannel{Name: name, Status: status}
		if err := db.Create(ch).Error; err != nil {
			t.Fatalf("写入渠道 %s 失败: %v", name, err)
		}
		return ch
	}

	activeA := newChannel("active-a", model.ChannelStatusActive)
	disabledB := newChannel("disabled-b", model.ChannelStatusDisabled)
	circuitC := newChannel("circuit-c", model.ChannelStatusCircuitOpen)
	activeD := newChannel("active-d", model.ChannelStatusActive)
	deletedE := newChannel("deleted-e", model.ChannelStatusActive)
	if err := db.Delete(&testChannel{}, deletedE.ID).Error; err != nil {
		t.Fatalf("软删渠道失败: %v", err)
	}

	createModel := func(channelID int64, name, status string, createdAt time.Time) *testModel {
		m := &testModel{
			ChannelID: channelID,
			Name:      name,
			Status:    status,
			CreatedAt: createdAt,
		}
		if err := db.Create(m).Error; err != nil {
			t.Fatalf("写入模型 %s 失败: %v", name, err)
		}
		return m
	}

	// 同名模型在两个可用渠道：应去重，created_at 取最早
	createModel(activeA.ID, "gpt-4o", model.ModelStatusActive, base.Add(2*time.Hour))
	createModel(activeD.ID, "gpt-4o", model.ModelStatusActive, base.Add(1*time.Hour))
	// 渠道不可用：不应出现
	createModel(disabledB.ID, "gpt-4o-mini", model.ModelStatusActive, base)
	createModel(circuitC.ID, "claude-3", model.ModelStatusActive, base)
	createModel(deletedE.ID, "gemini-pro", model.ModelStatusActive, base)
	// 模型自身 disabled：不应出现
	createModel(activeA.ID, "deepseek-chat", model.ModelStatusDisabled, base)
	// 模型软删：不应出现
	deletedModel := createModel(activeA.ID, "kimi", model.ModelStatusActive, base)
	if err := db.Delete(&testModel{}, deletedModel.ID).Error; err != nil {
		t.Fatalf("软删模型失败: %v", err)
	}

	rows, err := repo.ListEnabledModels(ctx)
	if err != nil {
		t.Fatalf("ListEnabledModels 返回错误: %v", err)
	}

	if len(rows) != 1 {
		t.Fatalf("模型数量 = %d, 期望 1，实际 %+v", len(rows), rows)
	}
	if rows[0].Name != "gpt-4o" {
		t.Fatalf("模型名 = %q, 期望 gpt-4o", rows[0].Name)
	}
	if !rows[0].CreatedAt.Equal(base.Add(time.Hour)) {
		t.Fatalf("created_at = %s, 期望 %s", rows[0].CreatedAt, base.Add(time.Hour))
	}
}

// TestChannelRepoListEnabledModelsEmpty 验证无可用模型时返回空切片而非 nil
func TestChannelRepoListEnabledModelsEmpty(t *testing.T) {
	db := newChannelModelsTestDB(t)
	repo := NewChannelRepo(db)

	rows, err := repo.ListEnabledModels(context.Background())
	if err != nil {
		t.Fatalf("ListEnabledModels 返回错误: %v", err)
	}
	if rows == nil {
		t.Fatal("期望返回空切片, 实际为 nil")
	}
	if len(rows) != 0 {
		t.Fatalf("模型数量 = %d, 期望 0", len(rows))
	}
}

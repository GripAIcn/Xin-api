package postgresql

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"Xin-api/internal/model"
)

// newExchangeRateTestDB 使用内存 SQLite 构建隔离的测试库
func newExchangeRateTestDB(t *testing.T, migrate bool) *gorm.DB {
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

	if migrate {
		if err := db.AutoMigrate(&model.ExchangeRate{}); err != nil {
			t.Fatalf("迁移汇率表失败: %v", err)
		}
	}
	return db
}

func newExchangeRateTestRepo(t *testing.T) (ExchangeRateRepo, *gorm.DB) {
	t.Helper()
	db := newExchangeRateTestDB(t, true)
	return NewExchangeRateRepo(db), db
}

func requireMoney(t *testing.T, got model.Money, want string) {
	t.Helper()
	expected := decimal.RequireFromString(want)
	if !got.Decimal.Equal(expected) {
		t.Fatalf("金额 = %s, 期望 %s", got.String(), want)
	}
}

func TestExchangeRateRepoGet(t *testing.T) {
	t.Run("记录不存在返回 nil,nil", func(t *testing.T) {
		repo, _ := newExchangeRateTestRepo(t)

		rate, err := repo.Get(context.Background(), model.CurrencyUSD, model.CurrencyCNY)
		if err != nil {
			t.Fatalf("Get 返回错误: %v", err)
		}
		if rate != nil {
			t.Fatalf("rate = %+v, 期望 nil", rate)
		}
	})

	t.Run("查询币种大小写与空格不敏感", func(t *testing.T) {
		repo, db := newExchangeRateTestRepo(t)
		if err := db.Create(&model.ExchangeRate{
			FromCurrency: model.CurrencyUSD,
			ToCurrency:   model.CurrencyCNY,
			Rate:         model.NewMoney(decimal.RequireFromString("7.2")),
			Source:       model.ExchangeRateSourceManual,
		}).Error; err != nil {
			t.Fatalf("写入测试数据失败: %v", err)
		}

		rate, err := repo.Get(context.Background(), " usd ", "cny")
		if err != nil {
			t.Fatalf("Get 返回错误: %v", err)
		}
		if rate == nil {
			t.Fatal("rate 不应为 nil")
		}
		requireMoney(t, rate.Rate, "7.2")
	})

	t.Run("查询失败返回错误", func(t *testing.T) {
		db := newExchangeRateTestDB(t, false) // 未建表
		repo := NewExchangeRateRepo(db)

		if _, err := repo.Get(context.Background(), model.CurrencyUSD, model.CurrencyCNY); err == nil {
			t.Fatal("期望返回错误, 实际为 nil")
		}
	})
}

func TestExchangeRateRepoUpsert(t *testing.T) {
	t.Run("新增汇率", func(t *testing.T) {
		repo, db := newExchangeRateTestRepo(t)
		updatedBy := int64(7)

		err := repo.Upsert(context.Background(), &model.ExchangeRate{
			FromCurrency: " usd ",
			ToCurrency:   "cny",
			Rate:         model.NewMoney(decimal.RequireFromString("7.2")),
			Source:       model.ExchangeRateSourceManual,
			UpdatedBy:    &updatedBy,
		})
		if err != nil {
			t.Fatalf("Upsert 返回错误: %v", err)
		}

		rate, err := repo.Get(context.Background(), model.CurrencyUSD, model.CurrencyCNY)
		if err != nil || rate == nil {
			t.Fatalf("查询新增记录失败: rate=%+v err=%v", rate, err)
		}
		if rate.FromCurrency != model.CurrencyUSD || rate.ToCurrency != model.CurrencyCNY {
			t.Fatalf("币种对 = %s→%s, 期望 USD→CNY", rate.FromCurrency, rate.ToCurrency)
		}
		requireMoney(t, rate.Rate, "7.2")
		if rate.Source != model.ExchangeRateSourceManual {
			t.Fatalf("来源 = %s, 期望 manual", rate.Source)
		}
		if rate.UpdatedBy == nil || *rate.UpdatedBy != updatedBy {
			t.Fatalf("updated_by = %v, 期望 %d", rate.UpdatedBy, updatedBy)
		}

		var count int64
		if err := db.Model(&model.ExchangeRate{}).Count(&count).Error; err != nil {
			t.Fatalf("统计记录数失败: %v", err)
		}
		if count != 1 {
			t.Fatalf("记录数 = %d, 期望 1", count)
		}
	})

	t.Run("相同币种对冲突时更新且不新增记录", func(t *testing.T) {
		repo, db := newExchangeRateTestRepo(t)
		firstBy := int64(1)
		secondBy := int64(2)

		if err := repo.Upsert(context.Background(), &model.ExchangeRate{
			FromCurrency: model.CurrencyUSD,
			ToCurrency:   model.CurrencyCNY,
			Rate:         model.NewMoney(decimal.RequireFromString("7.2")),
			Source:       model.ExchangeRateSourceManual,
			UpdatedBy:    &firstBy,
		}); err != nil {
			t.Fatalf("首次 Upsert 失败: %v", err)
		}

		if err := repo.Upsert(context.Background(), &model.ExchangeRate{
			FromCurrency: model.CurrencyUSD,
			ToCurrency:   model.CurrencyCNY,
			Rate:         model.NewMoney(decimal.RequireFromString("7.12345678")),
			Source:       model.ExchangeRateSourceAuto,
			UpdatedBy:    &secondBy,
		}); err != nil {
			t.Fatalf("再次 Upsert 失败: %v", err)
		}

		rate, err := repo.Get(context.Background(), model.CurrencyUSD, model.CurrencyCNY)
		if err != nil || rate == nil {
			t.Fatalf("查询更新记录失败: rate=%+v err=%v", rate, err)
		}
		requireMoney(t, rate.Rate, "7.12345678")
		if rate.Source != model.ExchangeRateSourceAuto {
			t.Fatalf("来源 = %s, 期望 auto", rate.Source)
		}
		if rate.UpdatedBy == nil || *rate.UpdatedBy != secondBy {
			t.Fatalf("updated_by = %v, 期望 %d", rate.UpdatedBy, secondBy)
		}

		var count int64
		if err := db.Model(&model.ExchangeRate{}).Count(&count).Error; err != nil {
			t.Fatalf("统计记录数失败: %v", err)
		}
		if count != 1 {
			t.Fatalf("记录数 = %d, 期望 1", count)
		}
	})

	t.Run("写库失败返回错误", func(t *testing.T) {
		db := newExchangeRateTestDB(t, false) // 未建表
		repo := NewExchangeRateRepo(db)

		err := repo.Upsert(context.Background(), &model.ExchangeRate{
			FromCurrency: model.CurrencyUSD,
			ToCurrency:   model.CurrencyCNY,
			Rate:         model.NewMoney(decimal.RequireFromString("7.2")),
			Source:       model.ExchangeRateSourceManual,
		})
		if err == nil {
			t.Fatal("期望返回错误, 实际为 nil")
		}
	})
}

func TestExchangeRateRepoEnsureUSDCNYSeed(t *testing.T) {
	t.Run("缺失时插入初始汇率", func(t *testing.T) {
		repo, db := newExchangeRateTestRepo(t)

		err := repo.EnsureUSDCNYSeed(context.Background(), model.NewMoney(decimal.RequireFromString("7.2")))
		if err != nil {
			t.Fatalf("EnsureUSDCNYSeed 返回错误: %v", err)
		}

		rate, err := repo.Get(context.Background(), model.CurrencyUSD, model.CurrencyCNY)
		if err != nil || rate == nil {
			t.Fatalf("查询种子记录失败: rate=%+v err=%v", rate, err)
		}
		requireMoney(t, rate.Rate, "7.2")
		if rate.Source != model.ExchangeRateSourceManual {
			t.Fatalf("来源 = %s, 期望 manual", rate.Source)
		}

		var count int64
		if err := db.Model(&model.ExchangeRate{}).Count(&count).Error; err != nil {
			t.Fatalf("统计记录数失败: %v", err)
		}
		if count != 1 {
			t.Fatalf("记录数 = %d, 期望 1", count)
		}
	})

	t.Run("已存在时不覆盖管理员配置", func(t *testing.T) {
		repo, db := newExchangeRateTestRepo(t)
		updatedBy := int64(9)
		if err := repo.Upsert(context.Background(), &model.ExchangeRate{
			FromCurrency: model.CurrencyUSD,
			ToCurrency:   model.CurrencyCNY,
			Rate:         model.NewMoney(decimal.RequireFromString("7.35")),
			Source:       model.ExchangeRateSourceAuto,
			UpdatedBy:    &updatedBy,
		}); err != nil {
			t.Fatalf("预置汇率失败: %v", err)
		}

		if err := repo.EnsureUSDCNYSeed(context.Background(), model.NewMoney(decimal.RequireFromString("7.2"))); err != nil {
			t.Fatalf("EnsureUSDCNYSeed 返回错误: %v", err)
		}

		rate, err := repo.Get(context.Background(), model.CurrencyUSD, model.CurrencyCNY)
		if err != nil || rate == nil {
			t.Fatalf("查询记录失败: rate=%+v err=%v", rate, err)
		}
		requireMoney(t, rate.Rate, "7.35")
		if rate.Source != model.ExchangeRateSourceAuto {
			t.Fatalf("来源 = %s, 期望 auto", rate.Source)
		}

		var count int64
		if err := db.Model(&model.ExchangeRate{}).Count(&count).Error; err != nil {
			t.Fatalf("统计记录数失败: %v", err)
		}
		if count != 1 {
			t.Fatalf("记录数 = %d, 期望 1", count)
		}
	})

	t.Run("查询失败返回错误", func(t *testing.T) {
		db := newExchangeRateTestDB(t, false) // 未建表
		repo := NewExchangeRateRepo(db)

		if err := repo.EnsureUSDCNYSeed(context.Background(), model.NewMoney(decimal.RequireFromString("7.2"))); err == nil {
			t.Fatal("期望返回错误, 实际为 nil")
		}
	})
}

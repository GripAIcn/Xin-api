package postgresql

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"Xin-api/internal/model"
)

type ExchangeRateRepo interface {
	// Get 查询指定币种对的汇率；未找到返回 (nil, nil)
	Get(ctx context.Context, fromCurrency, toCurrency string) (*model.ExchangeRate, error)
	// Upsert 按 (from_currency, to_currency) 新增或更新汇率
	Upsert(ctx context.Context, rate *model.ExchangeRate) error
	// EnsureUSDCNYSeed 确保 USD→CNY 汇率存在；已存在时不覆盖人工修改
	EnsureUSDCNYSeed(ctx context.Context, initialRate model.Money) error
}

type exchangeRateRepo struct {
	db *gorm.DB
}

// NewExchangeRateRepo 创建汇率配置仓储
func NewExchangeRateRepo(db *gorm.DB) ExchangeRateRepo {
	return &exchangeRateRepo{db: db}
}

// normalizeCurrency 统一币种代码格式
func normalizeCurrency(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

func (r *exchangeRateRepo) Get(ctx context.Context, fromCurrency, toCurrency string) (*model.ExchangeRate, error) {
	from := normalizeCurrency(fromCurrency)
	to := normalizeCurrency(toCurrency)
	rate, err := gorm.G[model.ExchangeRate](r.db).
		Where("from_currency = ? AND to_currency = ?", from, to).
		First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &rate, nil
}

func (r *exchangeRateRepo) Upsert(ctx context.Context, rate *model.ExchangeRate) error {
	rate.FromCurrency = normalizeCurrency(rate.FromCurrency)
	rate.ToCurrency = normalizeCurrency(rate.ToCurrency)
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "from_currency"}, {Name: "to_currency"}},
		DoUpdates: clause.AssignmentColumns([]string{"rate", "source", "updated_by", "updated_at"}),
	}).Create(rate).Error
}

func (r *exchangeRateRepo) EnsureUSDCNYSeed(ctx context.Context, initialRate model.Money) error {
	existing, err := r.Get(ctx, model.CurrencyUSD, model.CurrencyCNY)
	if err != nil {
		return err
	}
	if existing != nil {
		// 已存在则保留当前值，避免覆盖管理员配置
		return nil
	}
	return gorm.G[model.ExchangeRate](r.db).Create(ctx, &model.ExchangeRate{
		FromCurrency: model.CurrencyUSD,
		ToCurrency:   model.CurrencyCNY,
		Rate:         initialRate,
		Source:       model.ExchangeRateSourceManual,
	})
}

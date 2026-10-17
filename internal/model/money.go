package model

import (
	"database/sql/driver"
	"strings"

	"github.com/shopspring/decimal"
)

// Money 金额类型：内部使用 decimal 精确计算与存储（numeric），
// 对外 JSON 仍保持数字（而非 decimal 默认的字符串），以兼容现有前端。
type Money struct {
	decimal.Decimal
}

// NewMoney 由 decimal 构造金额
func NewMoney(d decimal.Decimal) Money {
	return Money{Decimal: d}
}

// MoneyFromFloat 由 float64 构造金额（仅用于兼容旧入参/配置）
func MoneyFromFloat(f float64) Money {
	return Money{Decimal: decimal.NewFromFloat(f)}
}

// Value 实现 driver.Valuer，写入 numeric 列
func (m Money) Value() (driver.Value, error) {
	return m.Decimal.Value()
}

// Scan 实现 sql.Scanner，从 numeric 列读取
func (m *Money) Scan(src any) error {
	return m.Decimal.Scan(src)
}

// MarshalJSON 输出 JSON 数字；零值统一为 0
func (m Money) MarshalJSON() ([]byte, error) {
	if m.Decimal.IsZero() {
		return []byte("0"), nil
	}
	return []byte(m.Decimal.String()), nil
}

// UnmarshalJSON 接受 JSON 数字或字符串，空值/null 视为 0
func (m *Money) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	if s == "" || s == "null" {
		m.Decimal = decimal.Zero
		return nil
	}
	s = strings.Trim(strings.TrimSpace(strings.Trim(s, `"`)), " ")
	if s == "" {
		m.Decimal = decimal.Zero
		return nil
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return err
	}
	m.Decimal = d
	return nil
}

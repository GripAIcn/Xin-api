package model

import (
	"database/sql/driver"
	"fmt"
	"strconv"
	"strings"
)

// Int64Array 用于 PostgreSQL 的 bigint[] 数组列。
//
// 背景：GORM 会把 []int64 当成标量处理，而 pgx 在 INSERT ... RETURNING 时
// 以文本形式（如 {1,2,3}）回传 bigint[]，直接扫描到 []int64 会报
// `unsupported Scan, storing driver.Value type string into type *[]int64`。
// 实现 driver.Valuer / sql.Scanner 后即可正确读写。
type Int64Array []int64

// Value 将切片编码为 PG 数组字面量，例如 {1,2,3}
func (a Int64Array) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}

	var b strings.Builder
	b.WriteByte('{')
	for i, elem := range a {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatInt(elem, 10))
	}
	b.WriteByte('}')
	return b.String(), nil
}

// Scan 解析 PG 返回的数组字面量（pgx 以文本形式回传）
func (a *Int64Array) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*a = nil
		return nil
	case []int64:
		*a = Int64Array(append([]int64(nil), v...))
		return nil
	case string:
		return a.parse(v)
	case []byte:
		return a.parse(string(v))
	}
	return fmt.Errorf("model.Int64Array: 无法扫描类型 %T", src)
}

func (a *Int64Array) parse(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		*a = nil
		return nil
	}
	if len(s) < 2 || s[0] != '{' || s[len(s)-1] != '}' {
		return fmt.Errorf("model.Int64Array: 非法的数组字面量 %q", s)
	}

	body := strings.TrimSpace(s[1 : len(s)-1])
	if body == "" {
		*a = Int64Array{}
		return nil
	}

	parts := strings.Split(body, ",")
	elems := make(Int64Array, 0, len(parts))
	for _, part := range parts {
		part = strings.Trim(strings.TrimSpace(part), `"`)
		if part == "" || strings.EqualFold(part, "null") {
			continue
		}
		n, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return fmt.Errorf("model.Int64Array: 无法解析数组元素 %q: %w", part, err)
		}
		elems = append(elems, n)
	}
	*a = elems
	return nil
}

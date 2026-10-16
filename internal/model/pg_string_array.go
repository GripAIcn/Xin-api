package model

import (
	"database/sql/driver"
	"fmt"
	"strings"
)

// StringArray 用于 PostgreSQL 数组列（varchar[] / text[]）。
//
// 背景：GORM 会把 []string 字段当成标量字符串绑定，导致写入 PG 数组列时报
// `malformed array literal: "chat" (SQLSTATE 22P02)`。实现 driver.Valuer / sql.Scanner
// 后，GORM 会原样透传该类型，pgx 再调用 Value() 得到合法的数组字面量（{chat}）。
type StringArray []string

// Value 将切片编码为 PG 数组字面量，例如 {"chat"}、{fast,cheap}
func (a StringArray) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}

	var b strings.Builder
	b.WriteByte('{')
	for i, elem := range a {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(quoteArrayElem(elem))
	}
	b.WriteByte('}')
	return b.String(), nil
}

// Scan 解析 PG 返回的数组字面量（pgx 以文本形式回传）
func (a *StringArray) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*a = nil
		return nil
	case []string:
		*a = StringArray(append([]string(nil), v...))
		return nil
	case string:
		elems, err := parseArrayLiteral(v)
		if err != nil {
			return err
		}
		*a = StringArray(elems)
		return nil
	case []byte:
		elems, err := parseArrayLiteral(string(v))
		if err != nil {
			return err
		}
		*a = StringArray(elems)
		return nil
	}
	return fmt.Errorf("model.StringArray: 无法扫描类型 %T", src)
}

// quoteArrayElem 按 PG 数组元素规则加引号（含特殊字符或 NULL 关键字时需转义）
func quoteArrayElem(s string) string {
	if needsArrayQuote(s) {
		var b strings.Builder
		b.WriteByte('"')
		for _, r := range s {
		if r == '"' || r == '\\' {
				b.WriteByte('\\')
			}
			b.WriteRune(r)
		}
		b.WriteByte('"')
		return b.String()
	}
	return s
}

func needsArrayQuote(s string) bool {
	if s == "" || strings.EqualFold(s, "null") {
		return true
	}
	return strings.ContainsAny(s, `{}",\ `) || strings.ContainsAny(s, "\t\n\r")
}

// parseArrayLiteral 解析形如 {a,"b,c",NULL} 的 PG 数组字面量
func parseArrayLiteral(s string) ([]string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	if len(s) < 2 || s[0] != '{' || s[len(s)-1] != '}' {
		return nil, fmt.Errorf("model.StringArray: 非法的数组字面量 %q", s)
	}

	body := s[1 : len(s)-1]
	if strings.TrimSpace(body) == "" {
		return []string{}, nil
	}

	var (
		elems  []string
		cur    strings.Builder
		quoted bool
		filled bool
	)
	for i := 0; i < len(body); i++ {
		c := body[i]
		switch {
		case c == '\\' && i+1 < len(body):
			i++
			cur.WriteByte(body[i])
			filled = true
		case c == '"':
			quoted = !quoted
			filled = true
		case c == ',' && !quoted:
			elems = append(elems, normalizeElem(cur.String(), filled))
			cur.Reset()
			filled = false
		default:
			cur.WriteByte(c)
			filled = true
		}
	}
	elems = append(elems, normalizeElem(cur.String(), filled))
	return elems, nil
}

// normalizeElem 将 NULL 元素归一为空串（业务上数组元素均为字符串标签）
func normalizeElem(v string, filled bool) string {
	if !filled || strings.EqualFold(v, "null") {
		return ""
	}
	return v
}

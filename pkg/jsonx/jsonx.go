// Package jsonx 提供数据面专用的 JSON 序列化工具。
//
// 标准库 encoding/json 默认会把字符串中的 <、>、& 转义为 \u003c、\u003e、\u0026，
// 这会导致代码、HTML、SQL 等内容经过网关后字面量被破坏。
// 本包统一关闭 HTML 转义，保证内容逐字节保真。
package jsonx

import (
	"bytes"
	"encoding/json"
)

// Marshal 序列化 JSON 且不做 HTML 转义（输出不含结尾换行）
func Marshal(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	out := buf.Bytes()
	// json.Encoder.Encode 会在末尾追加一个换行符，这里去掉以保持与 json.Marshal 一致
	if n := len(out); n > 0 && out[n-1] == '\n' {
		out = out[:n-1]
	}
	return out, nil
}

package jsonx

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMarshalDoesNotEscapeHTML(t *testing.T) {
	input := `<div class="x">&</div>`
	out, err := Marshal(map[string]string{"content": input})
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	got := string(out)
	for _, escaped := range []string{`\u003c`, `\u003e`, `\u0026`} {
		if strings.Contains(got, escaped) {
			t.Fatalf("unexpected escaped sequence %s in %s", escaped, got)
		}
	}
	var decoded map[string]string
	if err := json.Unmarshal(out, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if decoded["content"] != input {
		t.Fatalf("round-trip content = %q, want %q", decoded["content"], input)
	}
}

func TestMarshalNoTrailingNewline(t *testing.T) {
	out, err := Marshal(map[string]int{"a": 1})
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	if strings.HasSuffix(string(out), "\n") {
		t.Fatalf("unexpected trailing newline: %q", string(out))
	}
}

package tui

import (
	"strings"
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{500 * time.Nanosecond, "500ns"},
		{1500 * time.Nanosecond, "1.5µs"},
		{2500 * time.Microsecond, "2.5ms"},
		{1500 * time.Millisecond, "1.50s"},
	}

	for _, tt := range tests {
		got := formatDuration(tt.d)
		if got != tt.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		s      string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "hell…"},
		{"abc", 3, "abc"},
		{"abcd", 3, "ab…"},
	}

	for _, tt := range tests {
		got := truncate(tt.s, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.s, tt.maxLen, got, tt.want)
		}
	}
}

func TestVisibleRange(t *testing.T) {
	tests := []struct {
		cursor, total, height int
		wantStart, wantEnd    int
	}{
		// Fits in one page.
		{0, 5, 10, 0, 5},
		// Cursor at start.
		{0, 20, 10, 0, 10},
		// Cursor in the middle.
		{10, 20, 10, 5, 15},
		// Cursor near end.
		{18, 20, 10, 10, 20},
	}

	for _, tt := range tests {
		start, end := visibleRange(tt.cursor, tt.total, tt.height)
		if start != tt.wantStart || end != tt.wantEnd {
			t.Errorf("visibleRange(%d, %d, %d) = (%d, %d), want (%d, %d)",
				tt.cursor, tt.total, tt.height, start, end, tt.wantStart, tt.wantEnd)
		}
	}
}

func TestPadRow(t *testing.T) {
	result := padRow("hello", 10)
	if len(result) != 10 {
		t.Errorf("padRow should pad to width 10, got len %d", len(result))
	}
	if !strings.HasPrefix(result, "hello") {
		t.Error("padRow should preserve original content")
	}

	// Should not truncate if already wider.
	result = padRow("hello world!", 5)
	if result != "hello world!" {
		t.Error("padRow should not truncate")
	}
}

func TestSortedKeys(t *testing.T) {
	m := map[string]string{
		"zebra":    "1",
		"apple":    "2",
		"mango":    "3",
		"banana":   "4",
	}

	keys := sortedKeys(m)
	expected := []string{"apple", "banana", "mango", "zebra"}

	if len(keys) != len(expected) {
		t.Fatalf("expected %d keys, got %d", len(expected), len(keys))
	}

	for i, k := range keys {
		if k != expected[i] {
			t.Errorf("keys[%d] = %q, want %q", i, k, expected[i])
		}
	}
}

func TestRenderScrollHint_NoScrollNeeded(t *testing.T) {
	hint := renderScrollHint(0, 5, 10)
	if hint != "" {
		t.Errorf("should return empty when total <= height, got %q", hint)
	}
}

func TestRenderScrollHint_WithScroll(t *testing.T) {
	hint := renderScrollHint(5, 20, 10)
	if hint == "" {
		t.Error("should return hint when scrolling is possible")
	}
	if !strings.Contains(hint, "6/20") {
		t.Errorf("hint should contain position, got %q", hint)
	}
}

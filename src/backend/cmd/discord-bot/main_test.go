package main

import "testing"

func TestPageBounds(t *testing.T) {
	tests := []struct {
		name      string
		total     int
		page      int
		wantStart int
		wantEnd   int
	}{
		{name: "first page", total: 60, page: 0, wantStart: 0, wantEnd: 25},
		{name: "middle page", total: 60, page: 1, wantStart: 25, wantEnd: 50},
		{name: "last partial page", total: 60, page: 2, wantStart: 50, wantEnd: 60},
		{name: "page is clamped", total: 26, page: 99, wantStart: 25, wantEnd: 26},
		{name: "negative page is clamped", total: 26, page: -1, wantStart: 0, wantEnd: 25},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStart, gotEnd := pageBounds(tt.total, tt.page)
			if gotStart != tt.wantStart || gotEnd != tt.wantEnd {
				t.Fatalf("pageBounds(%d, %d) = (%d, %d), want (%d, %d)", tt.total, tt.page, gotStart, gotEnd, tt.wantStart, tt.wantEnd)
			}
		})
	}
}

func TestTruncateLabel(t *testing.T) {
	short := "Андрей"
	if got := truncateLabel(short); got != short {
		t.Fatalf("truncateLabel(%q) = %q", short, got)
	}
	long := "персонаж"
	for range 120 {
		long += "я"
	}
	if got := truncateLabel(long); len([]rune(got)) > 100 {
		t.Fatalf("truncateLabel returned %d runes, want <= 100", len([]rune(got)))
	}
}

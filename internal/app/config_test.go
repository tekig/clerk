package app

import "testing"

func TestToBytes(t *testing.T) {
	tests := []struct {
		in   string
		want int
	}{
		{"1KiB", 1024},
		{"5MiB", 5 * 1024 * 1024},
		{"128MiB", 128 * 1024 * 1024},
		{"2GiB", 2 * 1024 * 1024 * 1024},
	}
	for _, tt := range tests {
		got, err := toBytes(tt.in)
		if err != nil {
			t.Errorf("toBytes(%q): %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("toBytes(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}

	for _, in := range []string{"", "1024", "MiB", "5MB", "5 MiB", "1.5GiB"} {
		if _, err := toBytes(in); err == nil {
			t.Errorf("toBytes(%q): expected error", in)
		}
	}
}

package bytes

import "testing"

func Test_resize(t *testing.T) {
	tests := []struct {
		b []byte
		l int
	}{
		{
			b: make([]byte, 10),
			l: 10,
		},
		{
			b: make([]byte, 5),
			l: 10,
		},
		{
			b: make([]byte, 20),
			l: 10,
		},
	}
	for _, tt := range tests {
		got := Resize(tt.b, tt.l)
		if len(got) != tt.l {
			t.Errorf("resize() = %v, want %v", len(got), tt.l)
		}
	}
}

package xbytes

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
		{
			b: make([]byte, 0, 7),
			l: 8,
		},
	}
	for _, tt := range tests {
		got := Resize(tt.b, tt.l)
		if len(got) != tt.l {
			t.Errorf("resize() = %v, want %v", len(got), tt.l)
		}
	}
}

func Benchmark_resize(b *testing.B) {
	p := make([]byte, 0, 4*1024*1024)
	for range b.N {
		n := Resize(p, 8000)
		_ = n
	}
}

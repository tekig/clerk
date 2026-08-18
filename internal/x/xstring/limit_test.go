package xstring_test

import (
	"testing"

	"github.com/tekig/clerk/internal/x/xstring"
)

func TestLimitRight(t *testing.T) {
	tests := []struct {
		s    string
		n    int
		want string
	}{
		{
			"1234567890",
			5,
			"67890",
		}, {
			"1234567890",
			100,
			"1234567890",
		}, {
			"123",
			5,
			"123",
		}, {
			"🌍🚀",
			1,
			"🚀",
		}, {
			"🌍🚀",
			2,
			"🌍🚀",
		}, {
			"🌍🚀",
			3,
			"🌍🚀",
		},
	}
	for _, tt := range tests {
		got := xstring.LimitRight(tt.s, tt.n)
		if tt.want != got {
			t.Errorf("LimitRight() = %v, want %v", got, tt.want)
		}
	}
}

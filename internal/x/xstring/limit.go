package xstring

func LimitRight(s string, n int) string {
	if len(s) <= n {
		return s
	}

	runes := []rune(s)

	if len(runes) <= n {
		return s
	}

	return string(runes[len(runes)-n:])
}

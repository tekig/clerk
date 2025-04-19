package entity

import "errors"

type Error struct {
	Err  error
	Attr map[string]string
}

func (e Error) Error() (text string) {
	for k, v := range e.Attr {
		if text != "" {
			text += " "
		}
		text += k + "=" + v
	}
	return e.Err.Error() + " " + text
}

var (
	ErrNotFound = errors.New("not found")
)

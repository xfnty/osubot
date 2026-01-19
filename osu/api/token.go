package api

import (
	"time"
)

type Token struct {
	Value string
	ExpDate time.Time
}

func (t Token) Valid() bool {
	return t.Value != "" && time.Now().After(t.ExpDate)
}

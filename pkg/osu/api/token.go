package api

import (
	"io"
	"fmt"
	"time"
	"errors"
	"context"
	"strings"
	"net/http"
	"encoding/json"
)

var Host string

type Token struct {
	Access string
	ExpirationDate time.Time
}

func (t Token) Expired() bool {
	return time.Now().After(t.ExpirationDate)
}

func (t Token) TimeLeft() time.Duration {
	return t.ExpirationDate.Sub(time.Now())
}

func RequestToken(ctx context.Context, id, secret string) (Token, error) {
	payload := fmt.Sprintf(
		"client_id=%v&client_secret=%v&grant_type=client_credentials&scope=public",
		id,
		secret,
	)

	rq, e := http.NewRequestWithContext(ctx, "POST", Host + "/oauth/token", strings.NewReader(payload))
	if e != nil {
		return Token{}, e
	}

	rq.Header.Add("Accept", "application/json")
	rq.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	rp, e := (&http.Client{}).Do(rq)
	if e != nil {
		return Token{}, e
	}
	defer rp.Body.Close()

	b, e := io.ReadAll(rp.Body)
	if e != nil {
		return Token{}, e
	}

	var tr tokenResponse
	if e = json.Unmarshal(b, &tr); e != nil {
		return Token{}, e
	}
	if tr.Error != "" {
		return Token{}, errors.New(tr.ErrorDescription)
	}

	return Token{
		Access: tr.Access,
		ExpirationDate: time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second),
	}, nil
}

type tokenResponse struct {
	Error string            `json:"error"`
	ErrorDescription string `json:"error_description"`
	Access string           `json:"access_token"`
	ExpiresIn int           `json:"expires_in"`
}

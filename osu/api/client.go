package api

import (
	"io"
	"fmt"
	"time"
	"sync"
	"strings"
	"context"
	"net/http"
	"encoding/json"

	"osubot/log"
)

type Client struct {
	address, id, secret string
	mu sync.Mutex
	token token
}

func (c *Client) GetUserByName(ctx context.Context, username string) (u User, e error) {
	e = c.do(ctx, &u, "GET", fmt.Sprintf("/api/v2/users/@%v", username))
	return
}

func NewClient(address, id, secret string) Client {
	return Client{ address: address, id: id, secret: secret }
}

func (c *Client) do(ctx context.Context, response any, method, endpoint string) error {
	return c.doWithContent(ctx, response, "application/json", "", method, endpoint)
}

func (c *Client) doWithContent(
	ctx context.Context,
	response any,
	contentType,
	content,
	method,
	endpoint string,
) (e error) {
	var t token
	if t, e = c.ensureToken(ctx); e != nil {
		return
	}

	var rq *http.Request
	rq, e = http.NewRequestWithContext(ctx, method, c.address + endpoint, strings.NewReader(content))
	if e != nil {
		return
	}

	rq.Header.Set("Accept", "application/json")
	rq.Header.Set("Authorization", "Bearer " + t.value)
	rq.Header.Set("Content-Type", contentType)

	var rp *http.Response
	rp, e = httpClient.Do(rq)
	if e != nil {
		return
	}
	defer rp.Body.Close()

	log.Printf("%v %v (%v)", method, endpoint, rp.StatusCode)

	if rp.StatusCode != 200 {
		e = fmt.Errorf(rp.Status)
		return
	}

	var data []byte
	data, e = io.ReadAll(rp.Body)
	if e != nil {
		return
	}

	var errRp errorResponse
	if e = json.Unmarshal(data, &errRp); e != nil {
		return
	}
	if errRp.Error != "" {
		e = fmt.Errorf("%v: %v", errRp.Error, errRp.ErrorDescription)
		return
	}

	return json.Unmarshal(data, response)
}

func (c *Client) ensureToken(ctx context.Context) (token, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.token.valid() {
		return c.token, nil
	}

	content := fmt.Sprintf(
		"client_id=%v&client_secret=%v&grant_type=client_credentials&scope=public",
		c.id,
		c.secret,
	)
	rq, e := http.NewRequestWithContext(
		ctx,
		"POST",
		c.address + "/oauth/token",
		strings.NewReader(content),
	)
	if e != nil {
		return token{}, e
	}

	rq.Header.Set("Accept", "application/json")
	rq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rp, e := httpClient.Do(rq)
	if e != nil {
		return token{}, e
	}
	defer rp.Body.Close()

	if rp.StatusCode != 200 {
		return token{}, fmt.Errorf(rp.Status)
	}

	data, e := io.ReadAll(rp.Body)
	if e != nil {
		return token{}, e
	}

	var tokRp struct {
		errorResponse
		Access string `json:"access_token"`
		ExpiresIn int `json:"expires_in"`
	}
	if e = json.Unmarshal(data, &tokRp); e != nil {
		return token{}, e
	}
	if tokRp.Error != "" {
		return token{}, fmt.Errorf("%v: %v", tokRp.Error, tokRp.ErrorDescription)
	}

	c.token.value = tokRp.Access
	c.token.expdate = time.Now().Add(time.Duration(tokRp.ExpiresIn) * time.Second)

	log.Println("refreshed token")
	return c.token, nil
}

type token struct {
	value string
	expdate time.Time
}

func (t token) valid() bool {
	return t.value != "" && time.Now().After(t.expdate)
}

var httpClient http.Client

type errorResponse struct {
	Error string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

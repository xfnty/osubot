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
)

type Client struct {
	addr, id, secret string
	mu sync.Mutex
	httpClient http.Client
	token Token
}

func NewClient(addr, id, secret string) *Client {
	return &Client{ addr: addr, id: id, secret: secret }
}

func (c *Client) GetUserByName(ctx context.Context, name string) (u User, e error) {
	e = c.do(ctx, &u, "GET", fmt.Sprintf("/api/v2/users/@%v", name))
	return
}

func (c *Client) GetBeatmap(ctx context.Context, id int) (b Beatmap, e error) {
	e = c.do(ctx, &b, "GET", fmt.Sprintf("/api/v2/beatmaps/%v", id))
	return
}

func (c *Client) Token() Token {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.token
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
	var t Token
	if t, e = c.ensureToken(ctx); e != nil {
		return
	}

	var rq *http.Request
	rq, e = http.NewRequestWithContext(ctx, method, c.addr + endpoint, strings.NewReader(content))
	if e != nil {
		return
	}

	rq.Header.Set("Accept", "application/json")
	rq.Header.Set("Authorization", "Bearer " + t.Value)
	rq.Header.Set("Content-Type", contentType)

	var rp *http.Response
	rp, e = c.httpClient.Do(rq)
	if e != nil {
		return
	}
	defer rp.Body.Close()

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

func (c *Client) ensureToken(ctx context.Context) (Token, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.token.Valid() {
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
		c.addr + "/oauth/token",
		strings.NewReader(content),
	)
	if e != nil {
		return Token{}, e
	}

	rq.Header.Set("Accept", "application/json")
	rq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rp, e := c.httpClient.Do(rq)
	if e != nil {
		return Token{}, e
	}
	defer rp.Body.Close()

	if rp.StatusCode != 200 {
		return Token{}, fmt.Errorf(rp.Status)
	}

	data, e := io.ReadAll(rp.Body)
	if e != nil {
		return Token{}, e
	}

	var tokRp struct {
		errorResponse
		Access string `json:"access_token"`
		ExpiresIn int `json:"expires_in"`
	}
	if e = json.Unmarshal(data, &tokRp); e != nil {
		return Token{}, e
	}
	if tokRp.Error != "" {
		return Token{}, fmt.Errorf("%v: %v", tokRp.Error, tokRp.ErrorDescription)
	}

	c.token.Value = tokRp.Access
	c.token.ExpDate = time.Now().Add(time.Duration(tokRp.ExpiresIn) * time.Second)
	return c.token, nil
}

type errorResponse struct {
	Error string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

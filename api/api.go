package api

import (
	"io"
	"fmt"
	"time"
	"sync"
	"strings"
	"net/http"
	"osubot/util"
	"encoding/json"
)

type tokenResponse struct {
	Value string     `json:"access_token"`
	Lifetime int     `json:"expires_in"`
	TokenType string `json:"token_type"`
}

const (
	oauthTokenRefreshEndpoint = "https://osu.ppy.sh/oauth/token"
	v2RootEndpoint = "https://osu.ppy.sh/api/v2"
)

var client = &http.Client{ Timeout: 10 * time.Second }
var tokenRefreshPayload string
var token tokenResponse
var tokenMutex sync.Mutex
var tokenRefreshTicker *time.Ticker

func Init(id string, secret string) error {
	tokenRefreshPayload = fmt.Sprintf(
		"client_id=%v&client_secret=%v&grant_type=client_credentials&scope=public", 
		id, 
		secret,
	)

	if e := refreshToken(); e != nil {
		return e
	}

	tokenRefreshTicker = time.NewTicker(time.Duration(token.Lifetime) * time.Second)
	go func(){
		for _ = range tokenRefreshTicker.C {
			for e := refreshToken(); e != nil; {
				tokenRefreshTicker.Stop()
				util.StdoutLogger.Println("Failed to refresh API token:", e)
				time.Sleep(15)
			}
			tokenRefreshTicker.Reset(time.Duration(token.Lifetime) * time.Second)
		}
	}()

	return nil
}

func makeRequest(retval any, method string, payload string, path string, args ...interface{}) error {
	request, e := http.NewRequest(method, fmt.Sprintf(path, args...), strings.NewReader(payload))
	if e != nil {
		return e
	}

	request.Header.Set("Accept", "application/json")

	tokenMutex.Lock()
	if token.Value != "" {
		request.Header.Set("Authorization", "Bearer " + token.Value)
	}
	tokenMutex.Unlock()

	if payload != "" {
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	response, e := client.Do(request)
	if e != nil {
		return e
	}
	defer response.Body.Close()

	b, e := io.ReadAll(response.Body)
	if e != nil {
		return e
	}

	return json.Unmarshal(b, retval)
}

func refreshToken() error {
	var t tokenResponse
	if e := makeRequest(&t, "POST", tokenRefreshPayload, oauthTokenRefreshEndpoint); e != nil {
		return e
	}
	tokenMutex.Lock()
	token = t
	tokenMutex.Unlock()
	return nil
}

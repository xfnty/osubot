package cache

import (
	"os"
	"time"
	"sync"
	"path/filepath"
	"encoding/json"

	"osubot/pkg/osu/api"
)

func GetToken() (api.Token, error) {
	tokenFileInfo, e := os.Stat(tokenPath)

	token.mu.Lock()
	defer token.mu.Unlock()

	if token.value.Access == "" || (e != nil && tokenFileInfo.ModTime().After(token.lastUpdateTime)) {
		d, e := os.ReadFile(tokenPath)
		if e != nil {
			return api.Token{}, e
		}
		if e = json.Unmarshal(d, &token.value); e != nil {
			return api.Token{}, e
		}
		token.lastUpdateTime = time.Now()
	}

	return token.value, nil
}

func SetToken(t api.Token) error {
	b, e := json.MarshalIndent(t, "", "\t")
	if e != nil {
		return e
	}

	token.mu.Lock()
	defer token.mu.Unlock()

	os.MkdirAll(filepath.Dir(tokenPath), 0777)
	os.WriteFile(tokenPath, b, 0666)

	token.value = t
	token.lastUpdateTime = time.Now()

	return nil
}

const (
	tokenPath = "cache/token.json"
)

var token struct {
	value api.Token
	mu sync.Mutex
	lastUpdateTime time.Time
}

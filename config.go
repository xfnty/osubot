package osubot

import (
	"os"
	"encoding/json"
)

type Config struct {
	IRC struct {
		Addr string       `json:"address"`
		User string       `json:"username"`
		Pass string       `json:"password"`
		RateLimit float32 `json:"rate_limit"`
	} `json:"irc"`
	API struct {
		Addr string   `json:"address"`
		ID string     `json:"id"`
		Secret string `json:"secret"`
	} `json:"api"`
	HR struct {
		Enabled bool    `json:"enabled"`
		PrintQueue bool `json:"print_queue"`
	} `json:"host_rotation"`
	DC struct {
		Enabled bool `json:"enabled"`
		Range [2]float32 `json:"range"`
	} `json:"diffuclty_constraint"`
}

func (c *Config) LoadFile(path string) error {
	b, e := os.ReadFile(path)
	if e != nil {
		return e
	}
	if e = json.Unmarshal(b, c); e != nil {
		return e
	}
	return nil
}

func (c Config) SaveFile(path string) error {
	b, e := json.MarshalIndent(c, "", "\t")
	if e != nil {
		return e
	}
	return os.WriteFile(path, b, 0666)
}

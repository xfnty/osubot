package osubot

import (
	"os"
	"encoding/json"
)

type Cache struct {
	Lobby string `json:"lobby"`
}

func (c *Cache) LoadFile(path string) error {
	b, e := os.ReadFile(path)
	if e != nil {
		return e
	}
	if e = json.Unmarshal(b, c); e != nil {
		return e
	}
	return nil
}

func (c Cache) SaveFile(path string) error {
	b, e := json.MarshalIndent(c, "", "\t")
	if e != nil {
		return e
	}
	return os.WriteFile(path, b, 0666)
}

package api

import (
	"time"
)

type User struct {
	ID int              `json:"id"`
	Username string     `json:"username"`
	Avatar string       `json:"avatar_url"`
	CountryCode string  `json:"country_code"`
	IsOnline bool       `json:"is_online"`
	LastVisit time.Time `json:"last_visit"`
}

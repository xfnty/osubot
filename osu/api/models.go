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

type Mode string
const (
	ModeCatch = "fruits"
	ModeMania = "mania"
	ModeStandard = "osu"
	ModeTaiko = "taiko"
)

type BeatmapSet struct {
	ID int               `json:"id"`
	Creator string       `json:"creator"`
	Artist string        `json:"artist"`
	ArtistUnicode string `json:"artist_unicode"`
	Title string         `json:"title"`
	TitleUnicode string  `json:"title_unicode"`
}

type Beatmap struct {
	ID int                 `json:"id"`
	Name string            `json:"version"`
	Mode Mode              `json:"mode"`
	Length int             `json:"total_length"`
	Stars float32          `json:"difficulty_rating"`
	BeatmapSetID int       `json:"beatmapset_id"`
	BeatmapSet *BeatmapSet `json:"beatmapset"`
}

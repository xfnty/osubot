package api

type User struct {
	Id int             `json:"id"`
	Username string    `json:"username"`
	CountryCode string `json:"country_code"`
	IsOnline bool      `json:"is_online"`
}

type BeatmapSet struct {
	Id int               `json:"id"`
	Title string        `json:"title"`
	TitleUnicode string `json:"title_unicode"`
	Artist string        `json:"artist"`
	ArtistUnicode string `json:"artist_unicode"`
	Creator string       `json:"creator"`
}

type Beatmap struct {
	Name string            `json:"version"`
	Id int                 `json:"id"`
	BeatmapSetId int       `json:"beatmapset_id"`
	Stars float32          `json:"difficulty_rating"`
	BeatmapSet *BeatmapSet `json:"beatmapset"`
}

type Score struct {
	Rank string            `json:"rank"`
	Accuracy float32       `json:"accuracy"`
	Score int              `json:"score"`
	Pp *float32            `json:"pp"`
	MaxCombo int           `json:"max_combo"`
	IsPassed bool          `json:"passed"`
	Beatmap *Beatmap       `json:"beatmap"`
	BeatmapSet *BeatmapSet `json:"beatmapset"`
}

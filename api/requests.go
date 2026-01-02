package api

import (
	"slices"
)

type UserScoreQuery string
const (
	BestScores = "best"
	RecentScores = "recent"
)

const userScoresPagingLimit = 25

func GetUser(id interface{}) (user User, e error) {
	var idArg string
	switch v := id.(type) {
	case int:
		idArg = string(v)
	case string:
		idArg = "@" + v
	default:
		panic("GetUser(id) only accepts int or string as an argument")
	}

	e = makeRequest(&user, "GET", "", "%v/users/%v", v2RootEndpoint, idArg)
	return
}

func GetUserScores(id int, count int, query UserScoreQuery) (scores []Score, e error) {
	for offset := 0; count > 0; offset += userScoresPagingLimit {
		n := min(userScoresPagingLimit, count)
		var page []Score
		e = makeRequest(
			&page, 
			"GET", 
			"", 
			"%v/users/%v/scores/%v?limit=%v&offset=%v", 
			v2RootEndpoint, 
			id, 
			query,
			n,
			offset,
		)
		if e != nil {
			return
		}
		scores = slices.Concat(scores, page)
		count -= n
	}
	return
}

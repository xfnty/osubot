package api

type UserScoreQuery string
const (
	BestScores = "best"
	RecentScores = "recent"
)

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

func GetUserScores(id int, number int, query UserScoreQuery) (scores []Score, e error) {
	e = makeRequest(
		&scores, 
		"GET", 
		"", 
		"%v/users/%v/scores/%v?limit=%v", 
		v2RootEndpoint, 
		id, 
		query,
		number,
	)
	return
}

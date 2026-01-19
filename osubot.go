package main

import (
	"os"
	"fmt"
	"strings"
	"encoding/json"

	"osubot/irc"
)

type Bot struct {
	config config
	cache cache
	conn irc.Conn
	queue map[string]struct{}
}

func (b *Bot) OnAuthenticated() {
	if b.cache.Lobby != "" {
		fmt.Println("Attempting to rejoin", b.cache.Lobby)
		fmt.Fprintf(b.conn, "JOIN %v\n", b.cache.Lobby)
	} else {
		fmt.Println("Creating new lobby")
		fmt.Fprintf(b.conn, "PRIVMSG BanchoBot !mp make %v's game\n", b.config.IRC.User)
	}
}

func (b *Bot) OnAuthenticationError(e string) {
	fmt.Println(e)
	b.conn.Close()
}

func (b *Bot) OnJoined(lobby string, players []string) {
	for _, u := range players {
		b.queue[u] = struct{}{}
	}

	if b.cache.Lobby != lobby {
		fmt.Fprintf(b.conn, "PRIVMSG %v !mp password\n", lobby)
		fmt.Fprintf(b.conn, "PRIVMSG %v !mp size 8\n", lobby)
		fmt.Fprintf(b.conn, "PRIVMSG %v !mp invite %v\n", lobby, b.config.IRC.User)
	}

	b.cache.Lobby = lobby
	SaveJSON(cachePath, b.cache)

	fmt.Println("Joined", lobby)
}

func (b *Bot) OnJoinError(e string) {
	fmt.Println(e)

	if b.cache.Lobby != "" {
		b.cache.Lobby = ""
		SaveJSON(cachePath, b.cache)
		fmt.Fprintf(b.conn, "PRIVMSG BanchoBot !mp make %v's game\n", b.config.IRC.User)
	} else {
		b.conn.Close()
	}
}

func (b *Bot) OnLeft(lobby string) {
}

func (b *Bot) OnClosed(lobby string) {
	b.cache.Lobby = ""
	SaveJSON(cachePath, b.cache)
	b.conn.Close()
}

func (b *Bot) OnUserJoined(lobby, user string) {
	b.queue[user] = struct{}{}
	if len(b.queue) == 1 {
		fmt.Fprintf(b.conn, "PRIVMSG %v !mp host %v\n", lobby, user)
	}
	fmt.Println(user, "joined", lobby)
}

func (b *Bot) OnUserLeft(lobby, user string) {
	delete(b.queue, user)
	fmt.Println(user, "left", lobby)
	if len(b.queue) == 0 {
		fmt.Println("Closing the lobby")
		fmt.Fprintf(b.conn, "PRIVMSG %v !mp close\n", lobby)
	}
}

func (b *Bot) OnHostChanged(lobby, user string) {
	fmt.Println(user, "became the host")
}

func (b *Bot) OnBeatmapChanged(lobby, artist, title, difficulty string, id int) {
	fmt.Println("Selected", artist, "-", title)
}

func (b *Bot) OnAllPlayersReady(lobby string) {
	fmt.Println("all players ready")
	fmt.Fprintf(b.conn, "PRIVMSG %v !mp start", lobby)
}

func (b *Bot) OnMatchStarted(lobby string) {
	fmt.Println("Match started")
}

func (b *Bot) OnMatchFinished(lobby string) {
	fmt.Println("Match finished")
}

func (b *Bot) OnMatchAborted(lobby string) {
	fmt.Println("Match aborted")
}

func (b *Bot) OnUserMessage(lobby, user, message string) {
	fmt.Printf("%v: %v\n", user, message)
}

func (b *Bot) OnUserCommand(lobby, user, command string, args []string) {
	fmt.Printf("%v: %v(%v)\n", user, command, strings.Join(args, ", "))
}

func main() {
	var e error
	bot := Bot{ queue: make(map[string]struct{}) }

	fmt.Println("Loading", configPath)
	if e = LoadJSON(configPath, &bot.config); e != nil {
		panic(e)
	}

	fmt.Println("Loading", cachePath)
	LoadJSON(cachePath, &bot.cache)

	fmt.Println("Connecting to", bot.config.IRC.Addr)
	bot.conn, e = irc.Connect(bot.config.IRC.Addr)

	fmt.Println("Authenticating as", bot.config.IRC.User)
	fmt.Fprintf(bot.conn, "PASS %v\nNICK %v\n", bot.config.IRC.Pass, bot.config.IRC.User)

	for m, e := bot.conn.Get(); e == nil; m, e = bot.conn.Get() {
		irc.Dispatch(m, &bot)
	}
}

const (
	configPath = "config.json"
	cachePath = "cache.json"
)

type config struct {
	IRC struct {
		Addr string `json:"address"`
		User string `json:"username"`
		Pass string `json:"password"`
	} `json:"irc"`
}

type cache struct {
	Lobby string `json:"lobby"`
}

func LoadJSON(path string, object any) error {
	b, e := os.ReadFile(path)
	if e != nil {
		return e
	}
	if e = json.Unmarshal(b, object); e != nil {
		return e
	}
	return nil
}

func SaveJSON(path string, object any) error {
	b, e := json.MarshalIndent(object, "", "\t")
	if e != nil {
		return e
	}
	return os.WriteFile(path, b, 0666)
}

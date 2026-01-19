package main

import (
	"fmt"
	"context"
	"strings"
	"runtime/debug"

	"osubot"
	"osubot/osu/api"
	"osubot/osu/irc"
)

type Bot struct {
	config osubot.Config
	cache osubot.Cache
	conn irc.Conn
	api *api.Client
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
	b.cache.SaveFile(cachePath)

	fmt.Println("Joined", lobby)
}

func (b *Bot) OnJoinError(e string) {
	fmt.Println(e)

	if b.cache.Lobby != "" {
		b.cache.Lobby = ""
		b.cache.SaveFile(cachePath)
		fmt.Fprintf(b.conn, "PRIVMSG BanchoBot !mp make %v's game\n", b.config.IRC.User)
	} else {
		b.conn.Close()
	}
}

func (b *Bot) OnLeft(lobby string) {
}

func (b *Bot) OnClosed(lobby string) {
	b.cache.Lobby = ""
	b.cache.SaveFile(cachePath)
	b.conn.Close()
}

func (b *Bot) OnUserJoined(lobby, user string) {
	b.queue[user] = struct{}{}
	if len(b.queue) == 1 {
		fmt.Fprintf(b.conn, "PRIVMSG %v !mp host %v\n", lobby, user)
	}
	fmt.Println(user, "joined", lobby)

	go func(){
		u, e := b.api.GetUserByName(context.Background(), user)
		if e != nil {
			fmt.Println("Failed to get account info for", user)
			return
		}
		fmt.Printf("User %v: id=%v country=%v online=%v\n", u.Username, u.ID, u.CountryCode, u.IsOnline)
	}()
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

	go func(){
		bm, e := b.api.GetBeatmap(context.Background(), id)
		if e != nil {
			fmt.Println("Failed to get beatmap info for", id)
			return
		}
		fmt.Println(bm, bm.BeatmapSet)
	}()
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
	defer RecoverMain()

	var e error
	bot := Bot{ queue: make(map[string]struct{}) }

	fmt.Println("Loading", configPath)
	if e = bot.config.LoadFile(configPath); e != nil {
		panic(e)
	}

	bot.api = api.NewClient(bot.config.API.Addr, bot.config.API.ID, bot.config.API.Secret)

	fmt.Println("Loading", cachePath)
	bot.cache.LoadFile(cachePath)

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

func RecoverMain() {
	if r := recover(); r != nil {
		fmt.Printf("panic: %v\n\n%v", r, string(debug.Stack()))
		// fmt.Scan()
	}
}

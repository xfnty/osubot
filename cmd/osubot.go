package main

import (
	"os"
	"fmt"
	"slices"
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
	lobby string
	queue []string
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
	b.lobby = lobby
	b.queue = players

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
	b.queue = append(b.queue, user)

	if len(b.queue) == 1 {
		fmt.Fprintf(b.conn, "PRIVMSG %v !mp host %v\n", lobby, user)
	}

	go func(){
		u, e := b.api.GetUserByName(context.Background(), user)
		if e != nil {
			fmt.Println(e)
			return
		}
		fmt.Println(u)
	}()
}

func (b *Bot) OnUserLeft(lobby, user string) {
	i := slices.Index(b.queue, user)
	b.queue = slices.Concat(b.queue[:i], b.queue[i+1:])

	if len(b.queue) == 0 {
		fmt.Println("All players have left, closing the lobby")
		fmt.Fprintf(b.conn, "PRIVMSG %v !mp close\n", lobby)
	} else if b.config.AHR.Enabled && i == 0 {
		fmt.Fprintf(b.conn, "PRIVMSG %v !mp host %v\n", lobby, b.queue[0])
	}
}

func (b *Bot) OnHostChanged(lobby, user string) {
	if user != b.queue[0] && b.config.AHR.Enabled {
		fmt.Fprintf(b.conn, "PRIVMSG %v !mp host %v\n", lobby, b.queue[0])
		fmt.Fprintf(
			b.conn,
			"PRIVMSG %v %v, you can't transfer host to another player because host rotation is enabled. " +
			"You can ask %v to disable it.\n",
			lobby,
			b.queue[0],
			b.config.IRC.User,
		)
		return
	}
}

func (b *Bot) OnBeatmapChanged(lobby, artist, title, difficulty string, id int) {
	go func(){
		bm, e := b.api.GetBeatmap(context.Background(), id)
		if e != nil {
			fmt.Println(e)
			return
		}
		fmt.Println(bm, bm.BeatmapSet)
	}()
}

func (b *Bot) OnAllPlayersReady(lobby string) {
	fmt.Fprintf(b.conn, "PRIVMSG %v !mp start\n", lobby)
}

func (b *Bot) OnMatchStarted(lobby string) {
}

func (b *Bot) OnMatchFinished(lobby string) {
	if b.config.AHR.Enabled {
		b.rotateHost(lobby)
		if b.config.AHR.PrintQueue {
			b.printQueue(lobby)
		}
	}
}

func (b *Bot) OnMatchAborted(lobby string) {
}

func (b *Bot) OnUserMessage(lobby, user, message string) {
}

func (b *Bot) OnUserCommand(lobby, user, cmd string, args []string) {
	if cmd == "q" || cmd == "queue" {
		b.printQueue(lobby)
	} else if (cmd == "s" || cmd == "skip") && (user == b.queue[0] || user == b.config.IRC.User) {
		b.rotateHost(lobby)
		fmt.Println(user, "skipped host")
	} else if cmd == "ahr" && user == b.config.IRC.User {
		if len(args) == 0 {
			fmt.Fprintf(b.conn, "PRIVMSG %v AHR enabled: %v\n", lobby, b.config.AHR.Enabled)
		} else if len(args) == 1 && (args[0] == "on" || args[0] == "off") {
			if args[0] == "on" {
				b.config.AHR.Enabled = true
			} else {
				b.config.AHR.Enabled = false
			}
			fmt.Println("AHR enabled:", b.config.AHR.Enabled)
		} else {
			fmt.Fprintf(b.conn, "PRIVMSG %v Syntax: !ahr <on/off>\n", lobby)
		}
	} else if cmd == "help" || cmd == "info" {
		fmt.Fprintf(
			b.conn,
			"PRIVMSG %[1]v ┌ Info:\n" +
			"PRIVMSG %[1]v │    AHR enabled: %[2]v\n" +
			"PRIVMSG %[1]v │    Queue: %[3]v\n" +
			"PRIVMSG %[1]v │    [https://osu.ppy.sh/mp/%[4]v Match History]\n" +
			"PRIVMSG %[1]v │    [%[5]v Bot's Source Code]\n" +
			"PRIVMSG %[1]v ┌ Commands:\n" +
			"PRIVMSG %[1]v │    !q/queue – print host queue\n" +
			"PRIVMSG %[1]v │    !s/skip – skip host\n",
			lobby,
			b.config.AHR.Enabled,
			formatQueue(b.queue),
			lobby[4:],
			sourceRepository,
		)
	}
}

func (b *Bot) printQueue(lobby string) {
	fmt.Fprintf(
		b.conn,
		"PRIVMSG %v Queue: %v\n",
		lobby,
		formatQueue(b.queue),
	)
}

func (b *Bot) rotateHost(lobby string) {
	b.queue = slices.Concat(b.queue[1:], b.queue[:1])
	fmt.Fprintf(b.conn, "PRIVMSG %v !mp host %v\n", lobby, b.queue[0])
}

func formatQueue(queue []string) string {
	return strings.Join(slices.Concat(queue[1:], queue[:1]), ", ")
}

func main() {
	var e error
	b := Bot{}

	defer func(){
		if r := recover(); r != nil {
			m := fmt.Sprintf("panic: %v\n\n%v", r, string(debug.Stack()))
			os.WriteFile(crashPath, []byte(m), 0666)
			fmt.Println(m)
			fmt.Fprintf(b.conn, "PRIVMSG %v The bot has crashed: %v\n", b.config.IRC.User, r)
		}
	}()

	fmt.Println("Loading", configPath)
	if e = b.config.LoadFile(configPath); e != nil {
		panic(e)
	}

	b.api = api.NewClient(b.config.API.Addr, b.config.API.ID, b.config.API.Secret)

	fmt.Println("Loading", cachePath)
	b.cache.LoadFile(cachePath)

	fmt.Println("Connecting to", b.config.IRC.Addr)
	b.conn, e = irc.Connect(b.config.IRC.Addr)

	fmt.Println("Authenticating as", b.config.IRC.User)
	fmt.Fprintf(b.conn, "PASS %v\nNICK %v\n", b.config.IRC.Pass, b.config.IRC.User)

	for m, e := b.conn.Get(); e == nil; m, e = b.conn.Get() {
		irc.Dispatch(m, &b)
	}
}

const (
	configPath = "config.json"
	cachePath = "cache.json"
	crashPath = "crash.txt"
	sourceRepository = "https://github.com/xfnty/osubot"
)

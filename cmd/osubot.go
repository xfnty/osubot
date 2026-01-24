package main

import (
	"os"
	"fmt"
	"time"
	"sync"
	"slices"
	"context"
	"strings"
	"strconv"
	"os/signal"
	"runtime/debug"

	"osubot"
	"osubot/osu/api"
	"osubot/osu/irc"
)

type Player struct {
	Name string
	AutoSkip bool
}

type Bot struct {
	mu sync.Mutex
	config osubot.Config
	cache osubot.Cache
	conn irc.Conn
	api *api.Client
	lobby string
	queue []Player
	beatmap api.Beatmap
	matchInProgress bool
	matchStartTime time.Time
	mustDefineQueue bool
}

func (b *Bot) OnAuthenticated() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.cache.Lobby != "" {
		fmt.Println("Attempting to rejoin", b.cache.Lobby)
		b.conn.Send("JOIN", b.cache.Lobby)
	} else {
		fmt.Println("Creating new lobby")
		b.conn.Send("PRIVMSG", "BanchoBot", "!mp", "make", b.config.IRC.User + "'s game")
	}
}

func (b *Bot) OnAuthenticationError(e string) {
	panic(e)
}

func (b *Bot) OnJoined(lobby string, players []string) {
	fmt.Println("Joined lobby", lobby)

	b.mu.Lock()
	defer b.mu.Unlock()

	b.lobby = lobby

	b.queue = make([]Player, len(players), len(players))
	for i, name := range players {
		b.queue[i] = Player{ Name: name }
	}

	if b.cache.Lobby != lobby {
		b.conn.Send("PRIVMSG", lobby, "!mp", "password")
		b.conn.Send("PRIVMSG", lobby, "!mp", "mods", "Freemod")
		b.conn.Send("PRIVMSG", lobby, "!mp", "size", "8")
		b.conn.Send("PRIVMSG", lobby, "!mp", "invite", b.config.IRC.User)
	} else if len(players) > 1 {
		b.mustDefineQueue = true
		b.config.HR.Enabled = false
		fmt.Println("HR is disabled because the bot does not know what the host queue order was")
	}

	b.cache.Lobby = lobby
	b.cache.SaveFile(cachePath)
}

func (b *Bot) OnJoinError(e string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	fmt.Println(e)

	if b.cache.Lobby != "" {
		b.cache.Lobby = ""
		b.cache.SaveFile(cachePath)
		fmt.Println("Creating new lobby")
		b.conn.Send("PRIVMSG", "BanchoBot", "!mp", "make", b.config.IRC.User + "'s game")
	} else {
		b.conn.Close()
	}
}

func (b *Bot) OnLeft(lobby string) {
}

func (b *Bot) OnClosed(lobby string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.cache.Lobby = ""
	b.cache.SaveFile(cachePath)
	b.conn.Close()
}

func (b *Bot) OnUserJoined(lobby, user string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.queue = append(b.queue, Player{ Name: user })

	if len(b.queue) == 1 {
		b.conn.Send("PRIVMSG", lobby, "!mp", "host", user)
	}
}

func (b *Bot) OnUserLeft(lobby, user string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	i := slices.IndexFunc(b.queue, playerIndexFunc(user))
	b.queue = slices.Concat(b.queue[:i], b.queue[i+1:])

	if len(b.queue) == 0 {
		fmt.Println("All players have left, closing the lobby")
		b.conn.Send("PRIVMSG", lobby, "!mp", "close")
	} else if b.config.HR.Enabled && i == 0 {
		fmt.Printf("The host has left, transferring host to the next player (%v)\n", b.queue[0].Name)
		b.conn.Send("PRIVMSG", lobby, "!mp", "host", b.queue[0].Name)
	}

	if len(b.queue) <= 1 && b.mustDefineQueue {
		b.mustDefineQueue = false
		fmt.Println("HR queue can now be enabled")
	}
}

func (b *Bot) OnHostChanged(lobby, user string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if user != b.queue[0].Name && b.config.HR.Enabled && !b.mustDefineQueue {
		fmt.Println("Reverting illegal host transfer to", user)
		b.conn.Send("PRIVMSG", lobby, "!mp", "host", b.queue[0].Name)
		b.conn.Send(
			"PRIVMSG",
			lobby,
			fmt.Sprintf(
				"%v, you can't transfer host to another player because host rotation is enabled. " +
				"You can ask %v to disable it.",
				b.queue[0].Name,
				b.config.IRC.User,
			),
		)
		return
	}

	fmt.Println(user, "became the host")
}

func (b *Bot) OnBeatmapChanged(lobby, artist, title, difficulty string, id int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 3 * time.Second)
	defer cancel()
	bm, e := b.api.GetBeatmap(ctx, id)
	if e != nil {
		fmt.Println("Failed to fetch beatmap info:", e)
		return
	}

	if b.config.DC.Enabled {
		if bm.Stars < b.config.DC.Range[0] || bm.Stars > b.config.DC.Range[1] {
			if b.beatmap.ID != 0 {
				fmt.Printf(
					"Rejecting %v - %v [%v] %.2f*\n",
					bm.BeatmapSet.Artist,
					bm.BeatmapSet.Title,
					bm.Name,
					bm.Stars,
				)

				b.conn.Send("PRIVMSG", lobby, "!mp", "map", b.beatmap.ID, "0")

				var mapStatus string
				if bm.Stars < b.config.DC.Range[0] {
					mapStatus = fmt.Sprintf("too easy (%.2f<%v*)", bm.Stars, b.config.DC.Range[0])
				} else {
					mapStatus = fmt.Sprintf("too hard (%.2f>%v*)", bm.Stars, b.config.DC.Range[1])
				}
				b.conn.Send(
					"PRIVMSG",
					lobby,
					fmt.Sprintf(
						"%v, [https://osu.ppy.sh/beatmapsets/%v#osu/%v %v - %v [%v]] is %v. " +
						"You can ask %v to change the allowed difficulty range.",
						b.queue[0].Name,
						bm.BeatmapSetID,
						bm.ID,
						bm.BeatmapSet.Artist,
						bm.BeatmapSet.Title,
						bm.Name,
						mapStatus,
						b.config.IRC.User,
					),
				)
			} else {
				fmt.Printf(
					"%v - %v [%v] %.2f* should be rejected but there's no map to fallback to\n",
					bm.BeatmapSet.Artist,
					bm.BeatmapSet.Title,
					bm.Name,
					bm.Stars,
				)
			}
			return
		}
	}

	b.beatmap = bm
}

func (b *Bot) OnAllPlayersReady(lobby string) {
	b.conn.Send("PRIVMSG", lobby, "!mp", "start")
}

func (b *Bot) OnMatchStarted(lobby string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.matchInProgress = true
	b.matchStartTime = time.Now()
}

func (b *Bot) OnMatchFinished(lobby string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.matchInProgress = false

	if b.config.HR.Enabled && !b.mustDefineQueue {
		b.rotateHost(lobby)
		if b.config.HR.PrintQueue {
			b.printQueue(lobby)
		}
	}
}

func (b *Bot) OnMatchAborted(lobby string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.matchInProgress = false
}

func (b *Bot) OnUserMessage(lobby, user, message string) {
}

func (b *Bot) OnUserCommand(lobby, user, cmd string, args []string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if cmd == "q" || cmd == "queue" {
		if len(args) > 0 && user == b.config.IRC.User {
			newQueue := make([]Player, 0, len(b.queue))
			playersLeft := slices.Clone(b.queue)
			for _, nameApprox := range args {
				i := findOnePlayerByApprox(nameApprox, playersLeft)
				if i == -1 {
					fmt.Printf("No single player matches \"%v\" approximation.\n", nameApprox)
					continue
				}
				newQueue = append(newQueue, playersLeft[i])
				playersLeft = slices.Delete(playersLeft, i, i+1)
			}
			newQueue = slices.Concat(newQueue, playersLeft)
			if newQueue[0].Name != b.queue[0].Name {
				b.conn.Send("PRIVMSG", lobby, "!mp", "host", newQueue[0].Name)
			}
			b.queue = newQueue
			b.mustDefineQueue = false
		}
		b.printQueue(lobby)
	} else if (cmd == "s" || cmd == "skip") && (user == b.queue[0].Name || user == b.config.IRC.User) {
		b.rotateHost(lobby)
	} else if (cmd == "tl" || cmd == "timeleft") && b.matchInProgress && b.beatmap.ID != 0 {
		tl := b.matchStartTime.Add(time.Duration(b.beatmap.Length) * time.Second).Sub(time.Now())
		b.conn.Send(
			"PRIVMSG",
			lobby,
			fmt.Sprintf("Time left: %vm %vs", int(tl.Minutes()), int(tl.Seconds()) % 60),
		)
	} else if cmd == "hr" && user == b.config.IRC.User {
		if len(args) == 0 {
			if b.mustDefineQueue {
				b.conn.Send("PRIVMSG", lobby, "Host rotation is disabled until the queue is defined")
			} else {
				b.conn.Send(
					"PRIVMSG",
					lobby,
					"Host rotation is " + boolToEnabledDisabled(b.config.HR.Enabled),
				)
			}
		} else if len(args) == 1 && (args[0] == "on" || args[0] == "off") {
			if args[0] == "on" {
				if b.mustDefineQueue {
					fmt.Println("Attempted to enable HR without defining the queue")
					b.conn.Send("PRIVMSG", lobby, "Define the queue first using !q command")
					return
				}
				b.config.HR.Enabled = true
			} else {
				b.config.HR.Enabled = false
			}
			fmt.Println("HR", boolToEnabledDisabled(b.config.HR.Enabled))
		} else {
			b.conn.Send("PRIVMSG", lobby, "Syntax: !HR [on/off]")
		}
	} else if cmd == "dc" && user == b.config.IRC.User {
		if len(args) == 0 {
			b.conn.Send(
				"PRIVMSG",
				lobby,
				"Difficulty constraint is " + boolToEnabledDisabled(b.config.DC.Enabled),
			)
		} else if len(args) == 1 && (args[0] == "on" || args[0] == "off") {
			if args[0] == "on" {
				b.config.DC.Enabled = true
			} else {
				b.config.DC.Enabled = false
			}
			fmt.Println("DC", boolToEnabledDisabled(b.config.DC.Enabled))
		} else {
			b.conn.Send("PRIVMSG", lobby, "Syntax: !dc [on/off]")
		}
	} else if cmd == "dcr" && user == b.config.IRC.User {
		if len(args) == 0 {
			b.conn.Send(
				"PRIVMSG",
				lobby,
				fmt.Sprintf("Difficulty range is %v-%v*", b.config.DC.Range[0], b.config.DC.Range[1]),
			)
		} else if len(args) == 2 {
			rmin, e1 := strconv.ParseFloat(args[0], 32)
			rmax, e2 := strconv.ParseFloat(args[1], 32)
			if e1 == nil && e2 == nil {
				b.config.DC.Range[0], b.config.DC.Range[1] = float32(rmin), float32(rmax)
				fmt.Printf("Set DCR to %v-%v\n", b.config.DC.Range[0], b.config.DC.Range[1])
			} else {
				b.conn.Send("PRIVMSG", lobby, "Syntax: !dcr [min max]")
			}
		} else {
			b.conn.Send("PRIVMSG", lobby, "Syntax: !dcr [min max]")
		}
	} else if cmd == "pq" && user == b.config.IRC.User {
		if len(args) == 0 {
			b.conn.Send(
				"PRIVMSG",
				lobby,
				fmt.Sprintf("Print queue %v", boolToEnabledDisabled(b.config.HR.PrintQueue)),
			)
		} else if len(args) == 1 && (args[0] == "on" || args[0] == "off") {
			if args[0] == "on" {
				b.config.HR.PrintQueue = true
			} else {
				b.config.HR.PrintQueue = false
			}
			fmt.Println("PQ", boolToEnabledDisabled(b.config.HR.PrintQueue))
		} else {
			b.conn.Send("PRIVMSG", lobby, "Syntax: !pq on/off")
		}
	} else if cmd == "as" || cmd == "autoskip" {
		i := slices.IndexFunc(b.queue, playerIndexFunc(user))
		if i == -1 {
			fmt.Println(user, "is not in the queue!")
			return
		}
		b.queue[i].AutoSkip = !b.queue[i].AutoSkip
		b.conn.Send(
			"PRIVMSG",
			lobby,
			"Auto skip for",
			b.queue[i].Name,
			"is",
			boolToEnabledDisabled(b.queue[i].AutoSkip),
		)
	} else if cmd == "m" || cmd == "mirrors" {
		if b.beatmap.ID == 0 {
			fmt.Println("The bot couldn't get the beatmap info up to this point")
			return
		}
		b.conn.Send(
			"PRIVMSG",
			lobby,
			fmt.Sprintf(
				"[https://beatconnect.io/b/%[1]v BeatConnect] | " +
				"[https://nerinyan.moe/d/%[1]v NeriNyan] | " +
				"[https://catboy.best/d/%[1]v CatBoy]",
				b.beatmap.BeatmapSetID,
			),
		)
	}
}

func (b *Bot) printQueue(lobby string) {
	b.conn.Send("PRIVMSG", lobby, "Queue:", formatQueue(b.queue))
}

func (b *Bot) rotateHost(lobby string) {
	for i := 1; i < len(b.queue); i++ {
		if !b.queue[i].AutoSkip {
			b.queue = slices.Concat(b.queue[i:], b.queue[:i])
			b.conn.Send("PRIVMSG", lobby, "!mp", "host", b.queue[0].Name)
			return
		}
	}
}

func findOnePlayerByApprox(name string, players []Player) int {
	name = strings.ToLower(name)
	out := -1
	for i, player := range players {
		if strings.HasPrefix(strings.ToLower(player.Name), name) {
			if out != -1 {
				return -1
			}
			out = i
		}
	}
	return out
}

func formatQueue(queue []Player) string {
	names := make([]string,0, len(queue))
	for _, p := range queue {
		if !p.AutoSkip {
			names = append(names, p.Name)
		}
	}
	if len(names) > 0 {
		return strings.Join(slices.Concat(names[1:], names[:1]), ", ")
	}
	return "(empty)"
}

func boolToEnabledDisabled(flag bool) string {
	if flag {
		return "enabled"
	}
	return "disabled"
}

func playerIndexFunc(targetName string) func(Player)bool {
	return func(p Player)bool{ return p.Name == targetName }
}

func main() {
	var e error
	b := &Bot{}

	defer ReportPanic(b)

	fmt.Println("Loading", configPath)
	if e = b.config.LoadFile(configPath); e != nil {
		if os.IsNotExist(e) {
			fmt.Print("IRC username: ")
			fmt.Scanln(&b.config.IRC.User)
			fmt.Print("IRC password: ")
			fmt.Scanln(&b.config.IRC.Pass)
			fmt.Print("OAuth Client ID: ")
			fmt.Scanln(&b.config.API.ID)
			fmt.Print("OAuth Client Secret: ")
			fmt.Scanln(&b.config.API.Secret)
			b.config.IRC.Addr = "irc.ppy.sh:6667"
			b.config.IRC.RateLimit = 4
			b.config.API.Addr = "https://osu.ppy.sh"
			b.config.HR.Enabled = true
			b.config.DC.Range[1] = 10
			b.config.SaveFile(configPath)
		} else {
			panic(e)
		}
	}

	b.api = api.NewClient(b.config.API.Addr, b.config.API.ID, b.config.API.Secret)

	fmt.Println("Loading", cachePath)
	b.cache.LoadFile(cachePath)

	fmt.Println("Connecting to", b.config.IRC.Addr)
	b.conn, e = irc.Connect(b.config.IRC.Addr, b.config.IRC.RateLimit)

	fmt.Println("Authenticating as", b.config.IRC.User)
	b.conn.Send("PASS", b.config.IRC.Pass)
	b.conn.Send("NICK", b.config.IRC.User)

	errCh := make(chan error)
	go func(){
		defer ReportPanic(b)

		var e error
		var m irc.Msg
		for m, e = b.conn.Recv(); e == nil; m, e = b.conn.Recv() {
			if m.Cmd == "PING" {
				b.conn.Send("PONG")
				continue
			}
			irc.Dispatch(m, b)
		}
		errCh <- e
		close(errCh)
	}()

	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, os.Interrupt)

	select {
	case <-errCh:
		break
	case <-sigCh:
		var inp string
		fmt.Print("Close the lobby? [y/N]: ")
		fmt.Scanln(&inp)
		if inp == "y" {
			fmt.Println("Closing the lobby")
			b.conn.Send("PRIVMSG", b.lobby, "!mp", "close")
			<-errCh
		} else {
			b.conn.Close()
		}
	}
}

const (
	configPath = "config.json"
	cachePath = "cache.json"
	crashPath = "crash.txt"
	sourceRepository = "https://github.com/xfnty/osubot"
)

func ReportPanic(b *Bot) {
	if r := recover(); r != nil {
		m := fmt.Sprintf("panic: %v\n\n%v", r, string(debug.Stack()))
		os.WriteFile(crashPath, []byte(m), 0666)
		fmt.Println(m)
		b.conn.Send("PRIVMSG", b.config.IRC.User, "The bot has crashed:", r)
		os.Exit(1)
	}
}

package irc

import (
	"regexp"
	"strings"
	"strconv"

	"github.com/google/shlex"
)

type Dispatcher interface {
	OnAuthenticated()
	OnAuthenticationError(e string)
	OnJoined(lobby string, players []string)
	OnJoinError(e string)
	OnLeft(lobby string)
	OnClosed(lobby string)
	OnUserJoined(lobby, user string)
	OnUserLeft(lobby, user string)
	OnHostChanged(lobby, user string)
	OnBeatmapChanged(lobby, artist, title, difficulty string, id int)
	OnAllPlayersReady(lobby string)
	OnMatchStarted(lobby string)
	OnMatchFinished(lobby string)
	OnMatchAborted(lobby string)
	OnUserMessage(lobby, user, message string)
	OnUserCommand(lobby, user, command string, args []string)
}

func Dispatch(m Msg, d Dispatcher) {
	if m.Cmd == "001" {
		d.OnAuthenticated()
	} else if m.Cmd == "464" {
		d.OnAuthenticationError(m.Args[1])
	} else if m.Cmd == "403" {
		d.OnJoinError(m.Args[2])
	} else if m.Cmd == "353" {
		players := strings.Fields(m.Args[3])
		players = players[1:len(players)-1]
		d.OnJoined(m.Args[2], players)
	} else if m.Cmd == "PART" {
		d.OnLeft(m.Args[0])
	} else if m.Cmd == "PRIVMSG" {
		if m.Src == "BanchoBot" {
			if g := userJoinedRe.FindStringSubmatch(m.Args[1]); g != nil {
				d.OnUserJoined(m.Args[0], g[1])
			} else if g := userLeftRe.FindStringSubmatch(m.Args[1]); g != nil {
				d.OnUserLeft(m.Args[0], g[1])
			} else if g := hostChangedRe.FindStringSubmatch(m.Args[1]); g != nil {
				d.OnHostChanged(m.Args[0], g[1])
			} else if g := beatmapChangedRe.FindStringSubmatch(m.Args[1]); g != nil {
				if id, e := strconv.Atoi(g[4]); e == nil {
					d.OnBeatmapChanged(m.Args[0], g[1], g[2], g[3], id)
				}
			} else if m.Args[1] == "All players are ready" {
				d.OnAllPlayersReady(m.Args[0])
			} else if m.Args[1] == "The match has started!" {
				d.OnMatchStarted(m.Args[0])
			} else if m.Args[1] == "The match has finished!" {
				d.OnMatchFinished(m.Args[0])
			} else if m.Args[1] == "Aborted the match" {
				d.OnMatchAborted(m.Args[0])
			} else if m.Args[1] == "Closed the match" {
				d.OnClosed(m.Args[0])
			}
		} else {
			text := strings.Join(m.Args[1:], " ")
			if len(text) > 0 && text[0] == '!' {
				if args, e := shlex.Split(text); e == nil {
					d.OnUserCommand(m.Args[0], m.Src, args[0][1:], args[1:])
				}
			} else {
				d.OnUserMessage(m.Args[0], m.Src, text)
			}
		}
	}
}

var userJoinedRe, userLeftRe, hostChangedRe, beatmapChangedRe *regexp.Regexp

func init() {
	userJoinedRe, _ = regexp.Compile(`(\w+) joined in slot (\d+)\.`)
	userLeftRe, _ = regexp.Compile(`(\w+) left the game\.`)
	hostChangedRe, _ = regexp.Compile(`(\w+) became the host\.`)
	beatmapChangedRe, _ = regexp.Compile(`Beatmap changed to: (.+) - (.+) \[(.+)\] \(https://osu\.ppy\.sh/b/(\d+)\)`)
}

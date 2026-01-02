package irc

import (
	"regexp"
	"strconv"
	"strings"
	"github.com/google/shlex"
)

type LobbySimpleCallback func()
type LobbyChannelCallback func(channel string)
type LobbyPlayersCallback func(channel string, usernames []string)
type LobbyErrorCallback func(message string)
type LobbyUserCallback func(channel string, username string)
type LobbyBeatmapCallback func(channel string, id int)
type LobbyUserMessageCallback func(channel string, username string, message string)
type LobbyUserCommandCallback func(channel string, username string, command string, params []string)

type LobbyMessageDispatcher struct {
	Authenticated LobbySimpleCallback
	AuthenticationError LobbyErrorCallback
	JoinError LobbyErrorCallback
	JoinedLobby LobbyPlayersCallback
	LeftLobby LobbyChannelCallback
	LobbyClosed LobbyChannelCallback
	UserJoined LobbyUserCallback
	UserLeft LobbyUserCallback
	HostChanged LobbyUserCallback
	BeatmapChanged LobbyBeatmapCallback
	AllPlayersReady LobbyChannelCallback
	MatchStarted LobbyChannelCallback
	MatchFinished LobbyChannelCallback
	MatchAborted LobbyChannelCallback
	UserMessage LobbyUserMessageCallback
	UserCommand LobbyUserCommandCallback
	Owner string
}

var userJoinedRe *regexp.Regexp
var userLeftRe *regexp.Regexp
var hostChangedRe *regexp.Regexp
var beatmapChangedRe *regexp.Regexp

func init() {
	userJoinedRe, _ = regexp.Compile(`(\w+) joined in slot (\d+)\.`)
	userLeftRe, _ = regexp.Compile(`(\w+) left the game\.`)
	hostChangedRe, _ = regexp.Compile(`(\w+) became the host\.`)
	beatmapChangedRe, _ = regexp.Compile(`Beatmap changed to: (.+) - (.+) \[(.+)\] \(https://osu\.ppy\.sh/b/(\d+)\)`)
}

func (d LobbyMessageDispatcher) Dispatch(m Message) {
	if m.Command == "001" && d.Authenticated != nil {
		d.Authenticated()
	} else if m.Command == "464" && d.AuthenticationError != nil {
		e := m.Command
		if len(m.Params) == 2 {
			e = m.Params[1]
		}
		d.AuthenticationError(e)
	} else if m.Command == "403" && d.JoinError != nil {
		e := m.Command
		if len(m.Params) == 3 {
			e = m.Params[2]
		}
		d.JoinError(e)
	} else if m.Command == "353" && len(m.Params) == 4 && m.Params[0] == d.Owner && d.JoinedLobby != nil {
		usernames := strings.Fields(m.Params[3])
		usernames = usernames[1:len(usernames)-1]
		d.JoinedLobby(m.Params[2], usernames)
	} else if m.Command == "PART" && m.Source == d.Owner && d.LeftLobby != nil && len(m.Params) == 1 {
		d.LeftLobby(m.Params[0])
	} else if m.Command == "PRIVMSG" && len(m.Params) >= 2 {
		if m.Source == "BanchoBot" {
			if g := userJoinedRe.FindStringSubmatch(m.Params[1]); g != nil && d.UserJoined != nil {
				d.UserJoined(m.Params[0], g[1])
			} else if g := userLeftRe.FindStringSubmatch(m.Params[1]); g != nil && d.UserLeft != nil {
				d.UserLeft(m.Params[0], g[1])
			} else if g := hostChangedRe.FindStringSubmatch(m.Params[1]); g != nil && d.HostChanged != nil {
				d.HostChanged(m.Params[0], g[1])
			} else if g := beatmapChangedRe.FindStringSubmatch(m.Params[1]); g != nil && d.BeatmapChanged != nil {
				if id, e := strconv.Atoi(g[4]); e == nil {
					d.BeatmapChanged(m.Params[0], id)
				}
			} else if m.Params[1] == "All players are ready" && d.AllPlayersReady != nil {
				d.AllPlayersReady(m.Params[0])
			} else if m.Params[1] == "The match has started!" && d.MatchStarted != nil {
				d.MatchStarted(m.Params[0])
			} else if m.Params[1] == "The match has finished!" && d.MatchFinished != nil {
				d.MatchFinished(m.Params[0])
			} else if m.Params[1] == "Aborted the match" && d.MatchAborted != nil {
				d.MatchAborted(m.Params[0])
			} else if m.Params[1] == "Closed the match" && d.LobbyClosed != nil {
				d.LobbyClosed(m.Params[0])
			} 
		} else {
			text_message := strings.Join(m.Params[1:], " ")
			if len(text_message) > 0 {
				if text_message[0] == '!' && d.UserCommand != nil {
					if args, e := shlex.Split(text_message); e == nil {
						d.UserCommand(m.Params[0], m.Source, args[0][1:], args[1:])
					}
				} else if d.UserMessage != nil {
					d.UserMessage(m.Params[0], m.Source, text_message)
				}
			}
		}
	}
}

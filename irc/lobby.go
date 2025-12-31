package irc

import (
	"regexp"
	"strings"
	"github.com/google/shlex"
)

type LobbySimpleCallback func()
type LobbyChannelCallback func(channel string)
type LobbyErrorCallback func(message string)
type LobbyUserCallback func(channel string, username string)
type LobbyUserMessageCallback func(channel string, username string, message string)
type LobbyUserCommandCallback func(channel string, username string, command string, params []string)

type LobbyMessageDispatcher struct {
	Authenticated LobbySimpleCallback
	AuthenticationError LobbyErrorCallback
	JoinError LobbyErrorCallback
	JoinedLobby LobbyChannelCallback
	LeftLobby LobbyChannelCallback
	UserJoined LobbyUserCallback
	UserLeft LobbyUserCallback
	UserMessage LobbyUserMessageCallback
	UserCommand LobbyUserCommandCallback
	Owner string
}

var userJoinedRe *regexp.Regexp
var userLeftRe *regexp.Regexp

func init() {
	userJoinedRe, _ = regexp.Compile(`(\w+) joined in slot (\d+)\.`)
	userLeftRe, _ = regexp.Compile(`(\w+) left the game\.`)
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
	} else if m.Command == "MODE" && len(m.Params) == 3 && m.Params[1] == "+v" && m.Params[2] == d.Owner && d.JoinedLobby != nil {
		d.JoinedLobby(m.Params[0])
	} else if m.Command == "PART" && m.Source == d.Owner && d.LeftLobby != nil && len(m.Params) == 1 {
		d.LeftLobby(m.Params[0])
	} else if m.Command == "PRIVMSG" && len(m.Params) >= 2 {
		if m.Source == "BanchoBot" {
			if g := userJoinedRe.FindStringSubmatch(m.Params[1]); g != nil && d.UserJoined != nil {
				d.UserJoined(m.Params[0], g[1])
			} else if g := userLeftRe.FindStringSubmatch(m.Params[1]); g != nil && d.UserLeft != nil {
				d.UserLeft(m.Params[0], g[1])
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

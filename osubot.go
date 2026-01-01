package main

import (
	"os"
	"fmt"
	"slices"
	"strings"
	"os/signal"
	"osubot/irc"
	"osubot/util"
)

const SourceRepository = "https://github.com/xfnty/osubot"

var config *util.Config
var connection irc.Connection
var players []string
var skipVoters = make(map[string]struct{})
var startVoters = make(map[string]struct{})

func OnAuthenticated() {
	util.StdoutLogger.Println("Authenticated as", config.Credentials.Username)

	if config.SpecifiedChannel != ""  {
		fmt.Fprintf(connection, "JOIN %v\n", config.SpecifiedChannel)
	} else if config.SavedChannel != "" {
		fmt.Fprintf(connection, "JOIN %v\n", config.SavedChannel)
	} else {
		fmt.Fprintf(
			connection, 
			"PRIVMSG BanchoBot !mp make %v's game\n", 
			config.Credentials.Username,
		)
	}
}

func OnAuthenticationError(message string) {
	util.StdoutLogger.Println(message)
	connection.Close()
}

func OnJoinError(message string) {
	util.StdoutLogger.Println(message)
	
	if config.SpecifiedChannel == "" && config.SavedChannel != "" {
		config.SavedChannel = ""
		util.SaveChannel("")
		fmt.Fprintf(
			connection, 
			"PRIVMSG BanchoBot !mp make %v's game\n", 
			config.Credentials.Username,
		)
	} else {
		connection.Close()
	}
}

func OnJoinedLobby(channel string, usernames []string) {
	util.StdoutLogger.Println("Joined", channel)
	util.SaveChannel(channel)
	players = usernames

	if config.SpecifiedChannel == "" && config.SavedChannel == "" {
		fmt.Fprintf(connection, "PRIVMSG %v !mp password\n", channel)
	}

	if i := slices.Index(usernames, config.Credentials.Username); i == -1 {
		fmt.Fprintf(connection, "PRIVMSG %v !mp invite %v\n", channel, config.Credentials.Username)
	}
}

func OnLeftLobby(channel string) {
}

func OnLobbyClosed(channel string) {
	util.SaveChannel("")
	connection.Close()
}

func OnUserJoined(channel string, username string) {
	players = append(players, username)

	if len(players) == 1 {
		fmt.Fprintf(connection, "PRIVMSG %v !mp host %v\n", channel, username)
	}
}

func OnUserLeft(channel string, username string) {
	i := slices.Index(players, username)
	players = slices.Concat(players[:i], players[i+1:])

	if len(players) == 0 {
		fmt.Fprintf(connection, "PRIVMSG %v !mp close\n", channel)
	}
}

func OnHostChanged(channel string, username string) {
	i := slices.Index(players, username)
	if username != players[0] && config.HostRotation.Enabled {
		if config.HostRotation.AllowTransfers {
			phost, nhost := players[0], players[i]
			players = slices.Concat(
				players[i:i], 
				players[1:i], 
				players[i+1:len(players)], 
				players[0:0],
			)
			if config.HostRotation.ReportAllowedHostTransfers {
				fmt.Fprintf(
					connection, 
					"PRIVMSG %v %v gave host to %v, queue changed to: %v\n", 
					channel, 
					phost,
					nhost,
					strings.Join(slices.Concat(players[1:], players[:1]), ", "), 
				)
			}
		} else {
			fmt.Fprintf(connection, "PRIVMSG %v !mp host %v\n", channel, players[0])
			if config.HostRotation.ReportIllegalHostTransfers {
				fmt.Fprintf(
					connection, 
					"PRIVMSG %v %v, the host will be given to the next player automatically" +
					" because AHR is enabled.\n", 
					channel,
					players[0],
				)
			}
		}
	}
}

func OnBeatmapChanged(channel string, beatmap_id string) {
}

func OnAllPlayersReady(channel string) {
	fmt.Fprintf(connection, "PRIVMSG %v !mp start\n", channel)
}

func OnMatchStarted(channel string) {
	clear(skipVoters)
	clear(startVoters)
}

func OnMatchFinished(channel string) {
	if config.HostRotation.Enabled && len(players) > 1 {
		rotateHost()
		fmt.Fprintf(connection, "PRIVMSG %v !mp host %v\n", channel, players[0])
		if config.HostRotation.PrintQueueOnMatchEnd {
			printHostQueue(channel)
		}
	}
}

func OnMatchAborted(channel string) {
}

func OnUserMessage(channel string, username string, message string) {
	util.ChatLogger.Printf("%v: %v\n", username, message)
}

func OnUserCommand(channel string, username string, command string, params []string) {
	if command == "info" || command == "help" {
		fmt.Fprintf(
			connection,
			"PRIVMSG %[1]v ┌ Info:\n" +
			"PRIVMSG %[1]v │    AHR enabled: %[2]v\n" +
			"PRIVMSG %[1]v │    Players: %[3]v\n" +
			"PRIVMSG %[1]v │    Skip vote: %[4]v/%[5]v\n" +
			"PRIVMSG %[1]v │    Start vote: %[6]v/%[7]v\n" +
			"PRIVMSG %[1]v │    [https://osu.ppy.sh/mp/%[8]v Match History]\n" +
			"PRIVMSG %[1]v │    [%[9]v Bot's Source Code]\n" +
			"PRIVMSG %[1]v ┌ Commands:\n" +
			"PRIVMSG %[1]v │    !q – print host queue\n",
			channel,
			config.HostRotation.Enabled,
			strings.Join(players, ", "),
			len(skipVoters),
			int(float32(len(players)) * config.Voting.SkipVoteThreshold),
			len(startVoters),
			int(float32(len(players)) * config.Voting.StartVoteThreshold),
			channel[4:],
			SourceRepository,
		)
	} else if command == "q" && config.HostRotation.Enabled {
		printHostQueue(channel)
	} else if command == "skip" && config.HostRotation.Enabled && len(players) > 1 {
		if username == players[0] {
			rotateHost()
		} else {
			skipVoters[username] = struct{}{}
			players_required := int(float32(len(players)) * config.Voting.SkipVoteThreshold)
			if len(skipVoters) >= players_required {
				rotateHost()
			} else {
				fmt.Fprintf(
					connection, 
					"PRIVMSG %v Skip %v/%v\n", 
					channel, 
					len(skipVoters), 
					players_required,
				)
			}
		}
	} else if command == "start" {
		if username == players[0] {
			fmt.Fprintf(connection, "PRIVMSG %v !mp start\n", channel)
		} else {
			startVoters[username] = struct{}{}
			players_required := int(float32(len(players)) * config.Voting.StartVoteThreshold)
			if len(startVoters) >= players_required {
				fmt.Fprintf(connection, "PRIVMSG %v !mp start\n", channel)
			} else {
				fmt.Fprintf(
					connection, 
					"PRIVMSG %v Start %v/%v\n", 
					channel, 
					len(startVoters), 
					players_required,
				)
			}
		}
	}
}

func printHostQueue(channel string) {
	fmt.Fprintf(
		connection, 
		"PRIVMSG %v Queue: %v\n", 
		channel, 
		strings.Join(slices.Concat(players[1:], players[:1]), ", "), 
	)
}

func rotateHost() {
	players = slices.Concat(players[1:], players[:1])
	clear(skipVoters)
	clear(startVoters)
}

func main() {
	util.InitLogging()
	defer util.ShutdownLogging()

	var e error
	if config, e = util.LoadConfig(); e != nil {
		util.StdoutLogger.Fatalln(e)
	}
	util.StdoutLogger.Printf("Loaded \"%v\"", config.Path)

	irc.RateLimit = config.Server.RateLimit

	connection, e = irc.Connect(
		config.Server.Host,
		config.Server.Port,
		config.Credentials.Username,
		config.Credentials.Password,
	)
	if e != nil {
		util.StdoutLogger.Fatalln(e)
	}
	util.StdoutLogger.Println("Connected to", config.Server.Host)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	messages := make(chan irc.Message)

	go func(){
		for m, e := connection.Read(); e == nil; m, e = connection.Read() {
			messages <- m
		}
		close(messages)
	}()

	dispatcher := irc.LobbyMessageDispatcher{
		Owner: config.Credentials.Username,
		Authenticated: OnAuthenticated,
		AuthenticationError: OnAuthenticationError,
		JoinError: OnJoinError,
		JoinedLobby: OnJoinedLobby,
		LeftLobby: OnLeftLobby,
		LobbyClosed: OnLobbyClosed,
		UserJoined: OnUserJoined,
		UserLeft: OnUserLeft,
		HostChanged: OnHostChanged,
		BeatmapChanged: OnBeatmapChanged,
		AllPlayersReady: OnAllPlayersReady,
		MatchStarted: OnMatchStarted,
		MatchFinished: OnMatchFinished,
		MatchAborted: OnMatchAborted,
		UserMessage: OnUserMessage,
		UserCommand: OnUserCommand,
	}

	for running := true; running; {
		select {
		case m, open := <-messages:
			if !open {
				running = false
				break
			}
			if m.Command == "PING" {
				fmt.Fprintln(connection, "PONG")
				break
			}
			dispatcher.Dispatch(m)
		case <-interrupt:
			connection.Close()
			running = false
		}
	}
}

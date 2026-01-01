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

func OnAuthenticated() {
	util.StdoutLogger.Println("Authenticated as", config.Credentials.Username)

	if config.SpecifiedChannel != ""  {
		fmt.Fprintf(connection, "JOIN %v\n", config.SpecifiedChannel)
	} else if config.SavedChannel != "" {
		fmt.Fprintf(connection, "JOIN %v\n", config.SavedChannel)
	} else {
		fmt.Fprintf(connection, "PRIVMSG BanchoBot !mp make %v's game\n", config.Credentials.Username)
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
		fmt.Fprintf(connection, "PRIVMSG BanchoBot !mp make %v's game\n", config.Credentials.Username)
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
}

func OnUserLeft(channel string, username string) {
	i := slices.Index(players, username)
	players = slices.Concat(players[:i], players[i+1:])

	if len(players) == 0 {
		fmt.Fprintf(connection, "PRIVMSG %v !mp close\n", channel)
	}
}

func OnHostChanged(channel string, username string) {
}

func OnBeatmapChanged(channel string, beatmap_id string) {
}

func OnAllPlayersReady(channel string) {
	fmt.Fprintf(connection, "PRIVMSG %v !mp start\n", channel)
}

func OnMatchStarted(channel string) {
}

func OnMatchFinished(channel string) {
}

func OnMatchAborted(channel string) {
}

func OnUserMessage(channel string, username string, message string) {
	util.ChatLogger.Printf("%v: %v\n", username, message)
}

func OnUserCommand(channel string, username string, command string, params []string) {
	if command == "q" {
		fmt.Fprintf(
			connection, 
			"PRIVMSG %v Queue: %v\n", 
			channel, 
			strings.Join(slices.Concat(players[1:], players[:1]), ", "), 
		)
	} else if command == "info" || command == "help" {
		fmt.Fprintf(
			connection,
			"PRIVMSG %[1]v ┌ Info:\n" +
			"PRIVMSG %[1]v │    Players: %[2]v\n" +
			"PRIVMSG %[1]v │    [https://osu.ppy.sh/mp/%[3]v Match History]\n" +
			"PRIVMSG %[1]v │    [%[4]v Bot's Source Code]\n" +
			"PRIVMSG %[1]v ┌ Commands:\n" +
			"PRIVMSG %[1]v │    !q – print host queue\n",
			channel,
			strings.Join(players, ", "),
			channel[4:],
			SourceRepository,
		)
	}
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

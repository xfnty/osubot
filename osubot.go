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

var Config *util.Config
var Connection irc.Connection
var Channel string
var Players []string
var SkipVoters = make(map[string]struct{})
var StartVoters = make(map[string]struct{})

func OnAuthenticated() {
	util.StdoutLogger.Println("Authenticated as", Config.Credentials.Username)

	if Config.SpecifiedChannel != ""  {
		fmt.Fprintf(Connection, "JOIN %v\n", Config.SpecifiedChannel)
	} else if Config.SavedChannel != "" {
		fmt.Fprintf(Connection, "JOIN %v\n", Config.SavedChannel)
	} else {
		fmt.Fprintf(
			Connection, 
			"PRIVMSG BanchoBot !mp make %v's game\n", 
			Config.Credentials.Username,
		)
	}
}

func OnAuthenticationError(message string) {
	util.StdoutLogger.Println(message)
	Connection.Close()
}

func OnJoinError(message string) {
	util.StdoutLogger.Println(message)
	
	if Config.SpecifiedChannel == "" && Config.SavedChannel != "" {
		Config.SavedChannel = ""
		util.SaveChannel("")
		fmt.Fprintf(
			Connection, 
			"PRIVMSG BanchoBot !mp make %v's game\n", 
			Config.Credentials.Username,
		)
	} else {
		Connection.Close()
	}
}

func OnJoinedLobby(channel string, usernames []string) {
	Channel = channel
	Players = usernames

	util.StdoutLogger.Println("Joined", channel)
	util.SaveChannel(channel)

	if Config.SpecifiedChannel == "" && Config.SavedChannel == "" {
		fmt.Fprintf(Connection, "PRIVMSG %v !mp password\n", channel)
	}

	if i := slices.Index(usernames, Config.Credentials.Username); i == -1 {
		fmt.Fprintf(Connection, "PRIVMSG %v !mp invite %v\n", channel, Config.Credentials.Username)
	}
}

func OnLeftLobby(channel string) {
}

func OnLobbyClosed(channel string) {
	util.SaveChannel("")
	Connection.Close()
}

func OnUserJoined(channel string, username string) {
	Players = append(Players, username)

	if len(Players) == 1 {
		fmt.Fprintf(Connection, "PRIVMSG %v !mp host %v\n", channel, username)
	}
}

func OnUserLeft(channel string, username string) {
	i := slices.Index(Players, username)
	Players = slices.Concat(Players[:i], Players[i+1:])

	if len(Players) == 0 {
		fmt.Fprintf(Connection, "PRIVMSG %v !mp close\n", channel)
	}
}

func OnHostChanged(channel string, username string) {
	i := slices.Index(Players, username)
	if username != Players[0] && Config.HostRotation.Enabled {
		if Config.HostRotation.AllowTransfers {
			phost, nhost := Players[0], Players[i]
			Players = slices.Concat(
				Players[i:i], 
				Players[1:i], 
				Players[i+1:len(Players)], 
				Players[0:0],
			)
			if Config.HostRotation.ReportAllowedHostTransfers {
				fmt.Fprintf(
					Connection, 
					"PRIVMSG %v %v gave host to %v, queue changed to: %v\n", 
					channel, 
					phost,
					nhost,
					strings.Join(slices.Concat(Players[1:], Players[:1]), ", "), 
				)
			}
		} else {
			fmt.Fprintf(Connection, "PRIVMSG %v !mp host %v\n", channel, Players[0])
			if Config.HostRotation.ReportIllegalHostTransfers {
				fmt.Fprintf(
					Connection, 
					"PRIVMSG %v %v, the host will be given to the next player automatically" +
					" because AHR is enabled.\n", 
					channel,
					Players[0],
				)
			}
		}
	}
}

func OnBeatmapChanged(channel string, beatmap_id string) {
}

func OnAllPlayersReady(channel string) {
	fmt.Fprintf(Connection, "PRIVMSG %v !mp start\n", channel)
}

func OnMatchStarted(channel string) {
	clear(SkipVoters)
	clear(StartVoters)
}

func OnMatchFinished(channel string) {
	if Config.HostRotation.Enabled && len(Players) > 1 {
		rotateHost()
		fmt.Fprintf(Connection, "PRIVMSG %v !mp host %v\n", channel, Players[0])
		if Config.HostRotation.PrintQueueOnMatchEnd {
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
			Connection,
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
			Config.HostRotation.Enabled,
			strings.Join(Players, ", "),
			len(SkipVoters),
			int(float32(len(Players)) * Config.Voting.SkipVoteThreshold),
			len(StartVoters),
			int(float32(len(Players)) * Config.Voting.StartVoteThreshold),
			channel[4:],
			SourceRepository,
		)
	} else if command == "q" && Config.HostRotation.Enabled {
		printHostQueue(channel)
	} else if command == "skip" && Config.HostRotation.Enabled && len(Players) > 1 {
		if username == Players[0] {
			rotateHost()
		} else {
			SkipVoters[username] = struct{}{}
			Players_required := int(float32(len(Players)) * Config.Voting.SkipVoteThreshold)
			if len(SkipVoters) >= Players_required {
				rotateHost()
			} else {
				fmt.Fprintf(
					Connection, 
					"PRIVMSG %v Skip %v/%v\n", 
					channel, 
					len(SkipVoters), 
					Players_required,
				)
			}
		}
	} else if command == "start" {
		if username == Players[0] {
			fmt.Fprintf(Connection, "PRIVMSG %v !mp start\n", channel)
		} else {
			StartVoters[username] = struct{}{}
			Players_required := int(float32(len(Players)) * Config.Voting.StartVoteThreshold)
			if len(StartVoters) >= Players_required {
				fmt.Fprintf(Connection, "PRIVMSG %v !mp start\n", channel)
			} else {
				fmt.Fprintf(
					Connection, 
					"PRIVMSG %v Start %v/%v\n", 
					channel, 
					len(StartVoters), 
					Players_required,
				)
			}
		}
	}
}

func printHostQueue(channel string) {
	fmt.Fprintf(
		Connection, 
		"PRIVMSG %v Queue: %v\n", 
		channel, 
		strings.Join(slices.Concat(Players[1:], Players[:1]), ", "), 
	)
}

func rotateHost() {
	Players = slices.Concat(Players[1:], Players[:1])
	clear(SkipVoters)
	clear(StartVoters)
}

func main() {
	util.InitLogging()
	defer util.ShutdownLogging()

	var e error
	if Config, e = util.LoadConfig(); e != nil {
		util.StdoutLogger.Fatalln(e)
	}
	util.StdoutLogger.Printf("Loaded \"%v\"", Config.Path)

	irc.RateLimit = Config.Server.RateLimit

	Connection, e = irc.Connect(
		Config.Server.Host,
		Config.Server.Port,
		Config.Credentials.Username,
		Config.Credentials.Password,
	)
	if e != nil {
		util.StdoutLogger.Fatalln(e)
	}
	util.StdoutLogger.Println("Connected to", Config.Server.Host)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	messages := make(chan irc.Message)

	go func(){
		for m, e := Connection.Read(); e == nil; m, e = Connection.Read() {
			messages <- m
		}
		close(messages)
	}()

	dispatcher := irc.LobbyMessageDispatcher{
		Owner: Config.Credentials.Username,
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
				fmt.Fprintln(Connection, "PONG")
				break
			}
			dispatcher.Dispatch(m)
		case <-interrupt:
			b := make([]byte, 1)
			fmt.Print("Close the lobby? [Y/n]: ")
			n, _ := os.Stdin.Read(b)
			if n == 1 && b[0] != 'n' {
				fmt.Fprintf(connection, "PRIVMSG %v !mp close\n", Channel)
			}
			running = false
		}
	}
}

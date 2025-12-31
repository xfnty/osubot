package main

import (
	"os"
	"fmt"
	"os/signal"
	"osubot/irc"
	"osubot/util"
)

var config *util.Config
var connection irc.Connection

func OnAuthenticated() {
	util.StdoutLogger.Println("Authenticated as", config.Credentials.Username)
	if config.Channel == "" {
		fmt.Fprintf(connection, "PRIVMSG BanchoBot !mp make %v's game\n", config.Credentials.Username)
	} else {
		fmt.Fprintf(connection, "JOIN %v\n", config.Channel)
	}
}

func OnAuthenticationError(message string) {
	util.StdoutLogger.Println(message)
	connection.Close()
}

func OnJoinError(message string) {
	util.StdoutLogger.Println(message)
	connection.Close()
}

func OnJoinedLobby(channel string) {
	util.StdoutLogger.Println("Joined", channel)
	fmt.Fprintf(connection, "PRIVMSG %v !mp password\n", channel)
	fmt.Fprintf(connection, "PRIVMSG %v !mp invite %v\n", channel, config.Credentials.Username)
	fmt.Fprintf(connection, "PRIVMSG %v !mp close\n", channel)
}

func OnLeftLobby(channel string) {
	util.StdoutLogger.Println("Left", channel)
	connection.Close()
}

func OnUserJoined(channel string, username string) {
	util.StdoutLogger.Println(username, "joined")
	util.ChatLogger.Println(username, "joined")
}

func OnUserLeft(channel string, username string) {
	util.StdoutLogger.Println(username, "left")
	util.ChatLogger.Println(username, "left")
}

func OnUserMessage(channel string, username string, message string) {
	util.StdoutLogger.Printf("%v: %v\n", username, message)
	util.ChatLogger.Printf("%v: %v\n", username, message)
}

func OnUserCommand(channel string, username string, command string, params []string) {
	util.StdoutLogger.Printf("%v: %v %v\n", username, command, params)
}

func main() {
	util.InitLogging()
	defer util.ShutdownLogging()

	var e error
	if config, e = util.LoadConfig(); e != nil {
		util.StdoutLogger.Fatalln(e)
	}
	util.StdoutLogger.Printf("Loaded configuration from \"%v\"", config.Path)

	connection, e = irc.Connect(
		config.Server.Host,
		config.Server.Port,
		config.Credentials.Username,
		config.Credentials.Password,
	)
	if e != nil {
		util.StdoutLogger.Fatalln(e)
	}
	defer connection.Close()
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
		UserJoined: OnUserJoined,
		UserLeft: OnUserLeft,
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
    		running = false
    	}
    }
}

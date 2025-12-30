package main

import (
	"os"
	"errors"
	"strings"
	"os/signal"
	"osubot/irc"
	"osubot/util"
)

var config *util.Config
var authenticated bool
var connection irc.Connection

func dispatch(m irc.Message) error {
	if m.Command != "QUIT" {
		util.IrcLogger.Printf("%v: %v %v", m.Source, m.Command, strings.Join(m.Params, " "))
	}

	if m.Command == "001" {
		authenticated = true
		util.StdoutLogger.Println("Authenticated as", config.Credentials.Username)
	} else if m.Command == "464" {
		connection.Close()
		return errors.New(m.Params[1])
	}

	return nil
}

func main() {
    interrupt := make(chan os.Signal, 1)
    signal.Notify(interrupt, os.Interrupt)

    messages := make(chan irc.Message)

	util.InitLogging()
	defer util.ShutdownLogging()

	var e error
	config, e = util.LoadConfig()
	if e != nil {
		util.StdoutLogger.Fatalln(e)
	}

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

    go func(){
    	for m, e := connection.Read(); e == nil; m, e = connection.Read() {
    		messages <- m
    	}
    	close(messages)
    }()

    for running := true; running; {
    	select {
    	case m, open := <-messages:
    		if open {
    			if e = dispatch(m); e != nil {
			    	util.StdoutLogger.Fatalln(e)
    			}
    		} else {
    			if !authenticated {
			    	util.StdoutLogger.Fatalln("Connection closed")
			    }
    			running = false
    		}
    	case <-interrupt:
    		running = false
    	}
    }
}

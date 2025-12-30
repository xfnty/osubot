package util

import (
	"os"
	"fmt"
	"log"
	"time"
)

var chatLog *os.File
var ircLog *os.File

var StdoutLogger *log.Logger
var ChatLogger *log.Logger
var IrcLogger *log.Logger

func InitLogging() {
	var e error

	chatLog, e = os.OpenFile("chat.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if e != nil {
		panic(e)
	}

	ircLog, e = os.OpenFile("irc.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if e != nil {
		panic(e)
	}

	header := fmt.Sprintf("\nSession %v", time.Now())
	fmt.Fprintln(chatLog, header)
	fmt.Fprintln(ircLog, header)

	StdoutLogger = log.New(os.Stdout, "", log.Ltime|log.Lshortfile)
	ChatLogger = log.New(chatLog, "", log.Ltime)
	IrcLogger = log.New(ircLog, "", log.Ltime)
}

func ShutdownLogging() {
	chatLog.Close()
	ircLog.Close()
	chatLog = nil
	ircLog = nil
	StdoutLogger = nil
	ChatLogger = nil
	IrcLogger = nil
}

package main

import (
	"osubot/util"
)

func main() {
	util.InitLogging()
	defer util.ShutdownLogging()

	cfg, e := util.LoadConfig()
	if e != nil {
		util.StdoutLogger.Fatalln(e)
	}

	util.StdoutLogger.Println(cfg)
	util.ChatLogger.Println(cfg)
	util.IrcLogger.Println(cfg)
}

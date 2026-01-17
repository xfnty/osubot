package main

import (
	"os"
	"fmt"
	"context"
	"os/signal"
	"encoding/json"
	"runtime/debug"

	"github.com/pkg/browser"

	"osubot/log"
	"osubot/osu/api"
	"osubot/dashboard"
)

var config Config
var client api.Client
var owner api.User

func main() {
	defer Recover()

	var e error
	if config, e = LoadConfigFile("config.json"); e != nil {
		panic(e)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	client = api.NewClient(config.Api.Address, config.Api.Id, config.Api.Secret)

	if owner, e = client.GetUserByName(ctx, config.Irc.Username); e != nil {
		panic(e)
	}

	dashb := dashboard.Dashboard{ Address: config.Dashboard.Address, Owner: owner }
	go dashb.Run(ctx)
	if config.Dashboard.Open {
		browser.OpenURL(dashb.Link())
	}

	<-ctx.Done()
	log.Println("done")
}

func Recover() {
	if r := recover(); r != nil {
		m := []byte(fmt.Sprintf("panic: %v\n\n%v", r, string(debug.Stack())))
		os.WriteFile("crash.txt", m, 0666)
		os.Stderr.Write(m)
	}
}

type Config struct {
	Irc struct {
		Address string  `json:"address"`
		Username string `json:"username"`
		Password string `json:"password"`
	} `json:"irc"`
	Api struct {
		Address string   `json:"address"`
		Id string     `json:"id"`
		Secret string `json:"secret"`
	} `json:"api"`
	Dashboard struct {
		Address string `json:"address"`
		Open bool      `json:"open"`
	} `json:"dashboard"`
}

func LoadConfigFile(path string) (config Config, e error) {
	var b []byte
	b, e = os.ReadFile(path)
	if e != nil {
		return
	}
	e = json.Unmarshal(b, &config)
	return
}

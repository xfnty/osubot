package main

import (
	"os"
	"time"
	"context"
	"os/signal"
	"encoding/json"
	"runtime/debug"

	"github.com/pkg/browser"

	"osubot/pkg/osu/api"
	"osubot/cmd/bot/log"
	"osubot/cmd/bot/cache"
	"osubot/cmd/bot/dashboard"
)

type Config struct {
	Irc struct {
		Address string  `json:"address"`
		Username string `json:"username"`
		Password string `json:"password"`
	} `json:"irc"`
	Api struct {
		Host string   `json:"host"`
		Id string     `json:"id"`
		Secret string `json:"secret"`
	} `json:"api"`
	Dashboard struct {
		Address string `json:"address"`
		Open bool      `json:"open"`
	} `json:"dashboard"`
}

func LoadConfig(path string) (*Config, error) {
	b, e := os.ReadFile(path)
	if e != nil {
		return nil, e
	}
	c := &Config{}
	return c, json.Unmarshal(b, c)
}

func main() {
	defer func(){
		if r := recover(); r != nil {
			buildInfoStr := "No build info available."
			if b, bOk := debug.ReadBuildInfo(); bOk {
				buildInfoStr = b.String()
			}
			log.Crash.Printf(
				"panic: %v\n\n%v\n%v",
				r,
				string(debug.Stack()),
				buildInfoStr,
			)
		}
	}()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	config, e := LoadConfig("config.json")
	if e != nil {
		panic(e)
	}

	api.Host = config.Api.Host

	dashb := dashboard.Dashboard{ Address: config.Dashboard.Address }
	go dashb.Run(ctx)

	url := "http://" + dashb.Address
	log.Sys.Print("Serving dashboard on ", url)
	if config.Dashboard.Open {
		browser.OpenURL(url)
	}

	var token api.Token
	if token, e = cache.GetToken(); e != nil {
		requestContext, cancelRequest := context.WithTimeout(ctx, 3 * time.Second)
		defer cancelRequest()
		token, e = api.RequestToken(requestContext, config.Api.Id, config.Api.Secret)
		if e != nil {
			panic(e)
		}
		if e = cache.SetToken(token); e != nil {
			panic(e)
		}
		log.Sys.Printf("Requested new API token (expires in %v)", token.TimeLeft())
	} else {
		log.Sys.Printf("Loaded API token from cache (expires in %v)", token.TimeLeft())
	}

	<-ctx.Done()
}

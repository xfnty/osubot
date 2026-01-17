package dashboard

import (
	"fmt"
	"context"
	"net/http"

	"osubot/log"
	"osubot/osu/api"
)

type Dashboard struct {
	Address string
	Owner api.User
	server http.Server
	router *http.ServeMux
}

func (d Dashboard) Link() string {
	return "http://" + d.Address
}

func (d *Dashboard) Run(ctx context.Context) error {
	d.router = http.NewServeMux()
	d.router.HandleFunc("/{$}", d.handleIndex)

	d.server.Addr = d.Address
	d.server.Handler = d.router

	errorChan := make(chan error)
	go func(){
		if e := d.server.ListenAndServe(); e != http.ErrServerClosed {
			errorChan <- e
		}
		close(errorChan)
	}()

	log.Println("serving dashboard on ", d.Link())

	var e error
	select {
	case e = <-errorChan:
		break
	case <-ctx.Done():
		e = ctx.Err()
	}
	return e
}

func (d *Dashboard) handleIndex(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(rw, "Owner:<br><img src=\"%v\">", d.Owner.Avatar)
}

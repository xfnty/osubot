package dashboard

import (
	"fmt"
	"context"
	"net/http"
)

type Dashboard struct {
	Address string
	server http.Server
	router *http.ServeMux
}

func (d *Dashboard) handleIndex(rw http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(rw, "Welcome")
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

	var e error
	select {
	case e = <-errorChan:
		break
	case <-ctx.Done():
		e = ctx.Err()
	}
	return e
}

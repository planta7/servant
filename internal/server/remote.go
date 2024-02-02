// MIT Licensed
// Copyright (c) 2023 Roberto García <roberto@planta7.io>

package server

import (
	"github.com/charmbracelet/log"
	"github.com/localtunnel/go-localtunnel"
	"net"
	"net/http"
)

type remote struct {
	config Configuration
}

func newRemote(config Configuration) Server {
	return &remote{
		config: config,
	}
}

func (s *remote) Init(handler RequestHandler, httpHandler http.Handler) (*http.ServeMux, net.Listener, []string, error) {
	mux := http.NewServeMux()
	mux.Handle("/", handler.Handle(httpHandler))
	listener, err := localtunnel.Listen(localtunnel.Options{
		Log: log.StandardLog(),
	})
	if err != nil {
		return nil, nil, nil, err
	}
	return mux, listener, []string{listener.Addr().String()}, nil
}

func (s *remote) Start(server *http.Server, listener net.Listener) error {
	return server.Serve(listener)
}

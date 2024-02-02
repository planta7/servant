// MIT Licensed
// Copyright (c) 2023 Roberto García <roberto@planta7.io>

package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/planta7/servant/internal"
	"github.com/planta7/servant/internal/manager"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Type string

const (
	TypeLocal  Type = "local"
	TypeRemote Type = "remote"
)

type Configuration struct {
	Type       Type
	Path       string
	Host       string
	TLS        TLSRequest
	Port       int
	Expose     bool
	CORS       bool
	Launch     bool
	Auth       string
	DisableTUI bool
}

func (r *Configuration) WantsAutoTLS() bool {
	return r.TLS.Auto
}

func (r *Configuration) WantsTLS() bool {
	return r.WantsAutoTLS() || (r.TLS.CertFile != "" && r.TLS.KeyFile != "")
}

type TLSRequest struct {
	Auto     bool
	CertFile string
	KeyFile  string
}

type Server interface {
	Init(handler RequestHandler, httpHandler http.Handler) (*http.ServeMux, net.Listener, []string, error)
	Start(server *http.Server, listener net.Listener) error
}

type Servant struct {
	config    Configuration
	mux       *http.ServeMux
	listener  net.Listener
	server    Server
	addresses []string
}

func New(config Configuration) *Servant {
	var output manager.OutputManager
	if config.DisableTUI {
		log.Debug("Using Log output")
		output = manager.NewLogOutput()
	} else {
		log.Debug("Using TUI output")
		output = manager.NewTuiOutput()
	}

	var server Server
	var handler RequestHandler
	var httpHandler Handler
	if config.Type == TypeLocal {
		httpHandler = FileServer(http.Dir(config.Path))
		server = newLocal(config)
		handler = newLocalHandler(config, output)
		if config.Expose {
			server = newRemote(config)
		}
	} else {
		server = newRemote(config)
		handler = newProxyHandler(config, output)
	}
	mux, listener, addresses, err := server.Init(handler, httpHandler)
	if err != nil {
		log.Debug("net.Listen error", "error", fmt.Sprintf("%#v", err))
		if oErr, ok := err.(*net.OpError); ok {
			log.Fatal(oErr.Err.Error())
		}
	}

	output.Init(config.Path, addresses)

	return &Servant{
		config:    config,
		mux:       mux,
		listener:  listener,
		server:    server,
		addresses: addresses,
	}
}

func (s *Servant) Start() {
	server := &http.Server{
		Addr:    s.listener.Addr().String(),
		Handler: s.mux,
	}
	go s.start(server)

	stopCh, closeCh := createChannel()
	defer closeCh()
	log.Debug("Signal caught", "signal", <-stopCh)

	shutdown(context.Background(), server)
}

func (s *Servant) start(server *http.Server) {
	address := s.addresses[0]
	if s.config.Launch {
		log.Debug("Launching default browser", "url", address)
		err := internal.LaunchBrowser(address)
		if err != nil {
			log.Warn("Failed to launch", "error", err.Error())
		}
	}
	err := s.server.Start(server, s.listener)
	if errors.Is(err, http.ErrServerClosed) {
		log.Debug("Server closed")
	} else if err != nil {
		log.Fatal("Error listening for server", "err", err)
	}
}

func createChannel() (chan os.Signal, func()) {
	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	return stopCh, func() {
		close(stopCh)
	}
}

func shutdown(ctx context.Context, server *http.Server) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		panic(err)
	} else {
		log.Debug("Shutdown completed")
	}
}

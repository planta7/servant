// MIT Licensed
// Copyright (c) 2023 Roberto García <roberto@planta7.io>

package local

import (
	"context"
	"errors"
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/planta7/serve/internal"
	"github.com/planta7/serve/internal/network"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type Configuration struct {
	Path   string
	Host   string
	TLS    TLSConfiguration
	Port   int
	CORS   bool
	Launch bool
}

func (c *Configuration) WantsTLS() bool {
	return c.TLS.CertFile != "" && c.TLS.KeyFile != ""
}

type TLSConfiguration struct {
	CertFile string
	KeyFile  string
}

type serverValues struct {
	hosts  []string
	schema string
}

func (v *serverValues) getDefault() string {
	return fmt.Sprintf("%s://%s", v.schema, v.hosts[len(v.hosts)-1])
}

type Server struct {
	config Configuration
	values serverValues
}

func NewServer(config Configuration) *Server {
	return &Server{config: config}
}

func (s *Server) Start() {
	fullAddress := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	mux := http.NewServeMux()
	mux.Handle("/", s.handleRequest(http.FileServer(http.Dir(s.config.Path))))

	listener, err := net.Listen("tcp", fullAddress)
	if err != nil {
		log.Debug("net.Listen error", "error", fmt.Sprintf("%#v", err))
		if oErr, ok := err.(*net.OpError); ok {
			log.Fatal(oErr.Err.Error())
		}
	}
	server := &http.Server{
		Addr:    fullAddress,
		Handler: mux,
	}
	go s.start(server, listener)

	stopCh, closeCh := s.createChannel()
	defer closeCh()
	log.Debug("Signal caught", "signal", <-stopCh)

	s.shutdown(context.Background(), server)
}

func (s *Server) createChannel() (chan os.Signal, func()) {
	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	return stopCh, func() {
		close(stopCh)
	}
}

func (s *Server) start(server *http.Server, listener net.Listener) {
	s.resolveServerValues(listener)
	listenersValue := s.getListenersValue()
	log.Info(fmt.Sprintf("Serving %s at %s", s.config.Path, listenersValue))

	if s.config.Launch {
		url := s.values.getDefault()
		log.Debug("Launching default browser", "url", url)
		err := internal.LaunchBrowser(url)
		if err != nil {
			log.Warn("Invalid URL", "url", url)
		}
	}

	var err error
	if s.config.WantsTLS() {
		err = server.ServeTLS(listener, s.config.TLS.CertFile, s.config.TLS.KeyFile)
	} else {
		err = server.Serve(listener)
	}

	if errors.Is(err, http.ErrServerClosed) {
		log.Debug("Server closed")
	} else if err != nil {
		log.Fatal("Error listening for server", "err", err)
	}
}

func (s *Server) shutdown(ctx context.Context, server *http.Server) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		panic(err)
	} else {
		log.Debug("Shutdown completed")
	}
}

func (s *Server) resolveServerValues(listener net.Listener) {
	resolvedPort := s.config.Port
	if s.config.Port == 0 {
		resolvedPort = listener.Addr().(*net.TCPAddr).Port
	}
	var resolvedHosts []string
	if s.config.Host == "" {
		localIP, _ := network.LocalIP()
		resolvedHosts = append(resolvedHosts, fmt.Sprintf("127.0.0.1:%d", resolvedPort))
		resolvedHosts = append(resolvedHosts, fmt.Sprintf("%s:%d", localIP.String(), resolvedPort))
	} else {
		resolvedHosts = append(resolvedHosts, fmt.Sprintf("%s:%d", s.config.Host, resolvedPort))
	}
	resolvedSchema := "http"
	if s.config.WantsTLS() {
		resolvedSchema = "https"
	}
	s.values = serverValues{hosts: resolvedHosts, schema: resolvedSchema}
}

func (s *Server) getListenersValue() string {
	var listeners []string
	for _, h := range s.values.hosts {
		listeners = append(listeners, fmt.Sprintf("%s://%s", s.values.schema, h))
	}
	return strings.Join(listeners, ", ")
}

func (s *Server) getContentLength(header http.Header) string {
	value := header.Get(network.ContentLength)
	if value != "" {
		return fmt.Sprintf("(%s)", value)
	}
	return ""
}

func (s *Server) handleRequest(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lrw := network.NewLoggingResponseWriter(w)
		if s.config.CORS {
			w.Header().Set(network.AccessControlAllowOrigin, "*")
			w.Header().Set(network.AccessControlAllowMethods, "*")
		}
		h.ServeHTTP(lrw, r)
		contentLengthHeader := s.getContentLength(w.Header())
		statusStyle := internal.GetStyle(lrw.StatusCode)
		logLine := fmt.Sprintf("%s\t%s\t%s\t%s %s",
			r.RemoteAddr,
			statusStyle,
			r.Method,
			r.RequestURI,
			contentLengthHeader)
		log.Info(logLine)
	})
}

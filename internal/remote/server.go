// MIT Licensed
// Copyright (c) 2023 Roberto García <roberto@planta7.io>

package remote

import (
	"context"
	"errors"
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/localtunnel/go-localtunnel"
	"github.com/planta7/serve/internal/manager"
	"github.com/planta7/serve/internal/network"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

type ServerRequest struct {
	Port int
}

type Server struct {
	config ServerRequest
	output manager.OutputManager
	client *http.Client
}

func NewServer(config ServerRequest) *Server {
	return &Server{
		config: config,
		client: http.DefaultClient,
	}
}

func (s *Server) Start() {
	mux := http.NewServeMux()
	mux.Handle("/", s.handleRequest())

	listener, err := localtunnel.Listen(localtunnel.Options{
		Log: log.StandardLog(),
	})
	if err != nil {
		log.Debug("localtunnel.Listen error", "error", fmt.Sprintf("%#v", err))
		log.Fatal("Error creating listener for Localtunnel", "error", err.Error())
	}
	server := &http.Server{
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
	servingInfo := fmt.Sprintf("Serving port %d at %s", s.config.Port, listener.Addr().String())
	log.Info(servingInfo)

	log.Debug("Using Log output")
	s.output = manager.NewLogOutput()

	err := server.Serve(listener)
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

func (s *Server) handleRequest() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lrw := network.NewLoggingResponseWriter(w)

		proxyUrl := fmt.Sprintf("http://localhost:%d%s", s.config.Port, r.URL.Path)
		proxyReq, err := http.NewRequest(r.Method, proxyUrl, r.Body)
		if err != nil {
			log.Error("Error creating proxy request", err.Error())
			return
		}
		proxyRes, err := s.client.Do(proxyReq)
		if err != nil {
			log.Error("Error proxying request request", err.Error())
			if errors.Is(err, syscall.ECONNREFUSED) {
				errorMsg := fmt.Sprintf("SERVE: Connection to local port %d was refused, check that your server is up and running", s.config.Port)
				_, _ = w.Write([]byte(errorMsg))
			}
			return
		}
		for k, v := range proxyRes.Header {
			for _, hv := range v {
				w.Header().Add(k, hv)
			}
		}
		w.WriteHeader(proxyRes.StatusCode)
		_, err = io.Copy(w, proxyRes.Body)
		if err != nil {
			log.Error("Error copying body", err.Error())
			return
		}

		duration := time.Since(start)
		contentType := w.Header().Get(network.ContentType)
		stringContentLength := w.Header().Get(network.ContentLength)
		contentLength, _ := strconv.ParseInt(stringContentLength, 10, 64)
		unsignedContentLength := uint64(contentLength)

		request := &manager.Request{
			RemoteAddress: r.RemoteAddr,
			Url:           r.RequestURI,
			Method:        r.Method,
			Status:        lrw.StatusCode,
			Time:          &duration,
			Body:          &r.Body,
			ContentType:   contentType,
			ContentLength: unsignedContentLength,
		}
		s.output.Write(request)
	})
}

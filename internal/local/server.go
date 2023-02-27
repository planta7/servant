package local

import (
	"context"
	"errors"
	"fmt"
	"github.com/charmbracelet/log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"serve/internal/network"
	"syscall"
	"time"
)

type Configuration struct {
	Path   string
	Host   string
	Port   int
	Launch bool
}

type Server struct {
	config Configuration
}

func NewServer(config Configuration) *Server {
	return &Server{config: config}
}

func (s *Server) Start() {
	fullAddress := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	mux := http.NewServeMux()
	mux.Handle("/", s.logRequest(http.FileServer(http.Dir(s.config.Path))))

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
	go s.start(server, listener, s.config)

	stopCh, closeCh := s.createChannel()
	defer closeCh()
	log.Debug("Signal caught", "signal", <-stopCh)

	s.shutdown(context.Background(), server)
}

func (s *Server) getContentLength(header http.Header) string {
	value := header.Get("Content-Length")
	if value != "" {
		return fmt.Sprintf("(%s)", value)
	}
	return ""
}

func (s *Server) logRequest(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lrw := network.NewLoggingResponseWriter(w)
		h.ServeHTTP(lrw, r)
		contentLengthHeader := s.getContentLength(w.Header())
		logLine := fmt.Sprintf("%s\t%d\t%s\t%s %s", r.RemoteAddr, lrw.StatusCode, r.Method, r.RequestURI, contentLengthHeader)
		log.Info(logLine)
	})
}

func (s *Server) getListenerHost(listener net.Listener, config Configuration) string {
	resolvedPort := config.Port
	if config.Port == 0 {
		resolvedPort = listener.Addr().(*net.TCPAddr).Port
	}
	if config.Host == "" {
		localIP, _ := network.LocalIP()
		return fmt.Sprintf("http://127.0.0.1:%d and http://%s:%d", resolvedPort, localIP.String(), resolvedPort)
	}
	return fmt.Sprintf("http://%s:%d", config.Host, resolvedPort)
}

func (s *Server) createChannel() (chan os.Signal, func()) {
	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	return stopCh, func() {
		close(stopCh)
	}
}

func (s *Server) start(server *http.Server, listener net.Listener, config Configuration) {
	hostValue := s.getListenerHost(listener, config)
	log.Info(fmt.Sprintf("Serving %s at %s", config.Path, hostValue))
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

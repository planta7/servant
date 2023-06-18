// MIT Licensed
// Copyright (c) 2023 Roberto García <roberto@planta7.io>

package local

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	"github.com/planta7/serve/internal"
	"github.com/planta7/serve/internal/manager"
	"github.com/planta7/serve/internal/network"
	"github.com/planta7/serve/internal/tui"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type ServerRequest struct {
	Path   string
	Host   string
	TLS    TLSRequest
	Port   int
	CORS   bool
	Launch bool
	Auth   string
	TUI    bool
}

func (r *ServerRequest) WantsAutoTLS() bool {
	return r.TLS.Auto
}

func (r *ServerRequest) WantsTLS() bool {
	return r.WantsAutoTLS() || (r.TLS.CertFile != "" && r.TLS.KeyFile != "")
}

type TLSRequest struct {
	Auto     bool
	CertFile string
	KeyFile  string
}

type serverConfiguration struct {
	hosts       []string
	schema      string
	certificate serverCertificate
}

type serverCertificate struct {
	certFile string
	keyFile  string
}

func (v *serverConfiguration) getDefault() string {
	return fmt.Sprintf("%s://%s", v.schema, v.hosts[len(v.hosts)-1])
}

type Server struct {
	config   ServerRequest
	values   serverConfiguration
	requests *manager.RequestManager
	output   manager.OutputManager
}

func NewServer(config ServerRequest) *Server {
	return &Server{
		config:   config,
		requests: manager.NewRequestManager(),
	}
}

func (s *Server) Start() {
	fullAddress := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	fileServer := http.FileServer(http.Dir(s.config.Path))
	mux := http.NewServeMux()
	mux.Handle("/", s.handleRequest(fileServer))

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
	s.resolveServerConfiguration(listener)
	listenersValue := s.getListenersValue()
	servingInfo := fmt.Sprintf("Serving %s at %s", s.config.Path, listenersValue)
	log.Info(servingInfo)

	if s.config.TUI {
		log.Debug("Using TUI output")
		servingInfo = fmt.Sprintf("serve %s (%s) - TUI [experimental]\n%s",
			internal.ServeInfo.Version,
			internal.ServeInfo.GetShortCommit(),
			servingInfo)
		model := tui.NewModel(servingInfo)
		s.output = manager.NewTuiOutput(&model)

		go func() {
			if _, err := tea.NewProgram(model).Run(); err != nil {
				log.Fatal("Error running TUI", "error", err.Error())
			}
			p, _ := os.FindProcess(os.Getpid())
			_ = p.Signal(syscall.SIGTERM)
		}()
	} else {
		log.Debug("Using Log output")
		s.output = manager.NewLogOutput()
	}

	if s.config.Auth != "" {
		log.Debug("Using basic authentication")
	}

	if s.config.Launch {
		url := s.values.getDefault()
		log.Debug("Launching default browser", "url", url)
		err := internal.LaunchBrowser(url)
		if err != nil {
			log.Warn("Failed to launch", "error", err.Error())
		}
	}

	var err error
	if s.config.WantsTLS() {
		err = server.ServeTLS(listener, s.values.certificate.certFile, s.values.certificate.keyFile)
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

func (s *Server) resolveServerConfiguration(listener net.Listener) {
	port := s.config.Port
	if s.config.Port == 0 {
		port = listener.Addr().(*net.TCPAddr).Port
	}
	var hosts []string
	if s.config.Host == "" {
		localIP, _ := network.LocalIP()
		hosts = append(hosts, fmt.Sprintf("127.0.0.1:%d", port))
		hosts = append(hosts, fmt.Sprintf("%s:%d", localIP.String(), port))
	} else {
		hosts = append(hosts, fmt.Sprintf("%s:%d", s.config.Host, port))
	}
	var certFile, keyFile string
	schema := "http"
	if s.config.WantsTLS() {
		schema = "https"
		if s.config.WantsAutoTLS() {
			log.Debug("Generating self-signed certificate and key")
			certFile, keyFile = network.GenerateAutoTLS()
		} else {
			log.Debug("Using an external certificate and key", "cert", s.config.TLS.CertFile, "key", s.config.TLS.KeyFile)
			certFile, keyFile = s.config.TLS.CertFile, s.config.TLS.KeyFile
		}
	}
	s.values = serverConfiguration{
		hosts:  hosts,
		schema: schema,
		certificate: serverCertificate{
			certFile: certFile,
			keyFile:  keyFile,
		},
	}
}

func (s *Server) getListenersValue() string {
	var listeners []string
	for _, h := range s.values.hosts {
		listeners = append(listeners, fmt.Sprintf("%s://%s", s.values.schema, h))
	}
	return strings.Join(listeners, ", ")
}

func (s *Server) handleRequest(h http.Handler) http.Handler {
	requestHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lrw := network.NewLoggingResponseWriter(w)
		if s.config.CORS {
			w.Header().Set(network.AccessControlAllowOrigin, "*")
			w.Header().Set(network.AccessControlAllowMethods, "*")
		}
		h.ServeHTTP(lrw, r)
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
		s.requests.Add(request)
		s.output.Write(request)
	})

	if s.config.Auth != "" {
		return s.handleBasicAuth(requestHandler)
	} else {
		return requestHandler
	}
}

func (s *Server) handleBasicAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if ok {
			userPassword := strings.Split(s.config.Auth, ":")
			usernameHash := sha256.Sum256([]byte(username))
			passwordHash := sha256.Sum256([]byte(password))
			expectedUsernameHash := sha256.Sum256([]byte(userPassword[0]))
			expectedPasswordHash := sha256.Sum256([]byte(userPassword[1]))

			usernameMatch := subtle.ConstantTimeCompare(usernameHash[:], expectedUsernameHash[:]) == 1
			passwordMatch := subtle.ConstantTimeCompare(passwordHash[:], expectedPasswordHash[:]) == 1

			if usernameMatch && passwordMatch {
				next.ServeHTTP(w, r)
				return
			}
		}

		log.Warn("Basic auth not passed", r)
		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}

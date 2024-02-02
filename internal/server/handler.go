// MIT Licensed
// Copyright (c) 2023 Roberto García <roberto@planta7.io>

package server

import (
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/planta7/serve/internal/manager"
	"github.com/planta7/serve/internal/network"
	"io"
	"net/http"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type RequestHandler interface {
	Handle(h http.Handler) http.Handler
}

type localHandler struct {
	config   Configuration
	requests *manager.RequestManager
	output   manager.OutputManager
}

func newLocalHandler(config Configuration, output manager.OutputManager) RequestHandler {
	return &localHandler{
		config:   config,
		output:   output,
		requests: manager.NewRequestManager(),
	}
}

func (lh *localHandler) Handle(h http.Handler) http.Handler {
	requestHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lrw := network.NewLoggingResponseWriter(w)
		if lh.config.CORS {
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
		// TODO: channels
		lh.requests.Add(request)
		lh.output.Write(request)
	})

	if lh.config.Auth != "" {
		return lh.handleBasicAuth(requestHandler)
	} else {
		return requestHandler
	}
}

func (lh *localHandler) handleBasicAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if ok {
			userPassword := strings.Split(lh.config.Auth, ":")
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

type proxyHandler struct {
	localHandler
	client *http.Client
}

func newProxyHandler(config Configuration, output manager.OutputManager) RequestHandler {
	return &proxyHandler{
		localHandler: localHandler{
			config:   config,
			output:   output,
			requests: manager.NewRequestManager(),
		},
		client: http.DefaultClient,
	}
}

func (ph *proxyHandler) Handle(_ http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lrw := network.NewLoggingResponseWriter(w)

		proxyUrl := fmt.Sprintf("http://localhost:%d%s", ph.config.Port, r.URL.Path)
		proxyReq, err := http.NewRequest(r.Method, proxyUrl, r.Body)
		if err != nil {
			log.Error("Error creating proxy request", err.Error())
			return
		}
		proxyRes, err := ph.client.Do(proxyReq)
		if err != nil {
			log.Error("Error proxying request request", err.Error())
			if errors.Is(err, syscall.ECONNREFUSED) {
				errorMsg := fmt.Sprintf("SERVE: Connection to local port %d was refused, check that your server is up and running", ph.config.Port)
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
		// TODO: channels
		ph.requests.Add(request)
		ph.output.Write(request)
	})
}

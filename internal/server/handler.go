// MIT Licensed
// Copyright (c) 2023 Roberto García <roberto@planta7.io>

package server

import (
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/planta7/servant/internal/network"
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
	requests *Requests
	output   Output
}

func newLocalHandler(config Configuration, output Output) RequestHandler {
	return &localHandler{
		config:   config,
		output:   output,
		requests: NewRequestManager(),
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
		logRequest(start, w, r, lrw, lh.requests, lh.output)
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

func newProxyHandler(config Configuration, output Output) RequestHandler {
	return &proxyHandler{
		localHandler: localHandler{
			config:   config,
			output:   output,
			requests: NewRequestManager(),
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
				errorMsg := fmt.Sprintf("SERVANT: Connection to local port %d was refused, check that your server is up and running", ph.config.Port)
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

		logRequest(start, w, r, lrw, ph.requests, ph.output)
	})
}

func logRequest(
	start time.Time,
	w http.ResponseWriter,
	r *http.Request,
	lrw *network.LoggingResponseWriter,
	requests *Requests,
	output Output,
) {
	duration := time.Since(start)
	contentType := w.Header().Get(network.ContentType)
	stringContentLength := w.Header().Get(network.ContentLength)
	contentLength, _ := strconv.ParseInt(stringContentLength, 10, 64)
	unsignedContentLength := uint64(contentLength)

	request := &Request{
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
	requests.Add(request)
	output.Write(request)
}

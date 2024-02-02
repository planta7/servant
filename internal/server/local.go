// MIT Licensed
// Copyright (c) 2023 Roberto García <roberto@planta7.io>

package server

import (
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/planta7/servant/internal/network"
	"net"
	"net/http"
)

type serverCertificate struct {
	certFile string
	keyFile  string
}

type serverConfiguration struct {
	hosts       []string
	schema      string
	certificate serverCertificate
}

type local struct {
	config Configuration
	values serverConfiguration
}

func newLocal(config Configuration) Server {
	return &local{
		config: config,
	}
}

func (l *local) Init(handler RequestHandler, httpHandler http.Handler) (*http.ServeMux, net.Listener, []string, error) {
	fullAddress := fmt.Sprintf("%s:%d", l.config.Host, l.config.Port)
	mux := http.NewServeMux()
	mux.Handle("/", handler.Handle(httpHandler))
	listener, err := net.Listen("tcp", fullAddress)
	l.resolveConfig(listener)
	return mux, listener, l.addresses(), err
}

func (l *local) Start(server *http.Server, listener net.Listener) error {
	if l.config.Auth != "" {
		log.Debug("Using basic authentication")
	}

	if l.config.WantsTLS() {
		return server.ServeTLS(listener, l.values.certificate.certFile, l.values.certificate.keyFile)
	} else {
		return server.Serve(listener)
	}
}

func (l *local) resolveConfig(listener net.Listener) {
	port := l.config.Port
	if l.config.Port == 0 {
		port = listener.Addr().(*net.TCPAddr).Port
	}
	var hosts []string
	if l.config.Expose {
		hosts = append(hosts, listener.Addr().String())
	} else if l.config.Host == "" {
		localIP, _ := network.LocalIP()
		hosts = append(hosts, fmt.Sprintf("127.0.0.1:%d", port))
		hosts = append(hosts, fmt.Sprintf("%s:%d", localIP.String(), port))
	} else {
		hosts = append(hosts, fmt.Sprintf("%s:%d", l.config.Host, port))
	}
	var certFile, keyFile string
	schema := "http"
	if l.config.WantsTLS() {
		schema = "https"
		if l.config.WantsAutoTLS() {
			log.Debug("Generating self-signed certificate and key")
			certFile, keyFile = network.GenerateAutoTLS()
		} else {
			log.Debug("Using an external certificate and key", "cert", l.config.TLS.CertFile, "key", l.config.TLS.KeyFile)
			certFile, keyFile = l.config.TLS.CertFile, l.config.TLS.KeyFile
		}
	}
	l.values = serverConfiguration{
		hosts:  hosts,
		schema: schema,
		certificate: serverCertificate{
			certFile: certFile,
			keyFile:  keyFile,
		},
	}
}

func (l *local) addresses() []string {
	var addresses []string
	for _, h := range l.values.hosts {
		addresses = append(addresses, fmt.Sprintf("%s://%s", l.values.schema, h))
	}
	return addresses
}

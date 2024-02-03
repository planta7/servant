// MIT Licensed
// Copyright (c) 2023 Roberto García <roberto@planta7.io>

package server

import (
	"io"
	"time"
)

type Request struct {
	RemoteAddress string
	Url           string
	Method        string
	Status        int
	Time          *time.Duration
	Body          *io.ReadCloser
	ContentType   string
	ContentLength uint64
	// TODO: headers
}

type Requests struct {
	requests []*Request
}

func NewRequestManager() *Requests {
	return &Requests{requests: []*Request{}}
}

func (m *Requests) Add(request *Request) {
	m.requests = append(m.requests, request)
}

func (m *Requests) Find(url string) *Request {
	for _, r := range m.requests {
		if r.Url == url {
			return r
		}
	}
	return nil
}

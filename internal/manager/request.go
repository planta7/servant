// MIT Licensed
// Copyright (c) 2023 Roberto García <roberto@planta7.io>

package manager

import "time"

type Request struct {
	RemoteAddress string
	Url           string
	Method        string
	Status        int
	Time          *time.Duration
	Body          string
	ContentType   string
	ContentLength uint64
	// TODO: headers
}

type RequestManager struct {
	requests []*Request
}

func NewRequestManager() *RequestManager {
	return &RequestManager{requests: []*Request{}}
}

func (m *RequestManager) Add(request *Request) {
	m.requests = append(m.requests, request)
}

func (m *RequestManager) Find(url string) *Request {
	for _, r := range m.requests {
		if r.Url == url {
			return r
		}
	}
	return nil
}

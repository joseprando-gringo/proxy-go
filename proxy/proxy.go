package proxy

import "net/http"

type Proxy interface {
	HostId() string
	AppendAuth(string) string
	SetTargetHost(*http.Request)
	CacheResponse(string, *http.Response)
}

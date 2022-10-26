package main

import (
	"bytes"
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"strings"

	d "github.com/joseprando-gringo/proxy/datacube"
	p "github.com/joseprando-gringo/proxy/proxy"
)

// Hop-by-hop headers. These are removed when sent to the backend.
// http://www.w3.org/Protocols/rfc2616/rfc2616-sec13.html
var hopHeaders = []string{
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te", // canonicalized version of "TE"
	"Trailers",
	"Transfer-Encoding",
	"Upgrade",
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func delHopHeaders(header http.Header) {
	for _, h := range hopHeaders {
		header.Del(h)
	}
}

func appendHostToXForwardHeader(header http.Header, host string) {
	// If we aren't the first proxy retain prior
	// X-Forwarded-For information as a comma+space
	// separated list and fold multiple headers into one.
	if prior, ok := header["X-Forwarded-For"]; ok {
		host = strings.Join(prior, ", ") + ", " + host
	}
	header.Set("X-Forwarded-For", host)
}

type proxy struct {
}

var proxies = [1]p.Proxy{
	&d.DataCube{},
}

func ProxyBySchemeAndHost(scheme string, host string) p.Proxy {
	for _, v := range proxies {
		if strings.Contains(scheme+"://"+host, v.HostId()) {
			return v
		}
	}

	return nil
}

func RequestInterceptor(req *http.Request, proxyInstance p.Proxy) {
	reqBodyBytes, err := io.ReadAll(req.Body)
	req.Body.Close()
	if err != nil {
		log.Panic(err)
	}
	reqBodyString := string(reqBodyBytes)
	reqBodyString = proxyInstance.AppendAuth(reqBodyString)
	reqBodyBytesNew := []byte(reqBodyString)
	req.ContentLength = int64(len(reqBodyBytesNew))
	req.Body = io.NopCloser(bytes.NewReader(reqBodyBytesNew))
}

func ResponseInterceptor(resp *http.Response) {
	bodyBytes, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		log.Panic(err)
	}
	bodyString := string(bodyBytes)
	log.Println(bodyString)
	resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
}

func (p *proxy) ServeHTTP(wr http.ResponseWriter, req *http.Request) {
	log.Println(req.RemoteAddr, " ", req.Method, " ", req.URL)

	if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
		msg := "unsupported protocal scheme " + req.URL.Scheme
		http.Error(wr, msg, http.StatusBadRequest)
		log.Println(msg)
		return
	}

	proxyInstance := ProxyBySchemeAndHost(req.URL.Scheme, req.URL.Host)
	if proxyInstance == nil {
		log.Println(req.RemoteAddr, " ", http.StatusNotFound)
		wr.WriteHeader(http.StatusNotFound)
		return
	}

	// Lock - Checa o Lock da API (Enfileiramento por Chave (Ex: Placa))
	// Cache - Busca no cache, se não encontrar só coloca a chave na request

	// Codinomes para APIs - Troca o nome falso para o original
	// Validar se o proxy tem codinome, se tiver, faz essa troca aqui, senão manda o que veio do cliente
	proxyInstance.SetTargetHost(req)

	client := &http.Client{}

	//http: Request.RequestURI can't be set in client requests.
	//http://golang.org/src/pkg/net/http/client.go
	req.RequestURI = ""

	delHopHeaders(req.Header)

	if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		appendHostToXForwardHeader(req.Header, clientIP)
	}

	// [BEGIN] - Interceptador de Request
	RequestInterceptor(req, proxyInstance)
	// [END] - Interceptador de Request

	resp, err := client.Do(req)
	if err != nil {
		http.Error(wr, "Server Error", http.StatusInternalServerError)
		log.Panic("ServeHTTP:", err)
	}
	// defer resp.Body.Close()

	log.Println(req.RemoteAddr, " ", resp.Status)

	// [BEGIN] Interceptador da Response
	ResponseInterceptor(resp)
	defer resp.Body.Close()
	// [END] Interceptador da Response

	delHopHeaders(resp.Header)

	// Cachear a resposta (Cada proxy fornece a sua implementação)
	proxyInstance.CacheResponse("", resp)

	copyHeader(wr.Header(), resp.Header)
	wr.WriteHeader(resp.StatusCode)
	io.Copy(wr, resp.Body)
}

func main() {
	var addr = flag.String("addr", "127.0.0.1:8080", "The addr of the application.")
	flag.Parse()

	handler := &proxy{}

	log.Println("Starting proxy server on", *addr)
	if err := http.ListenAndServe(*addr, handler); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

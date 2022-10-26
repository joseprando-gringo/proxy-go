package datacube

import "net/http"

type DataCube struct {
}

func (*DataCube) HostId() string {
	return "http://datacube-api.gringo.com.vc"
}

func (*DataCube) AppendAuth(bodyAsString string) string {
	return bodyAsString + "&auth_token=8F6BEA6C-5D3B-4091-9664-BFC7B1445552"
}

func (*DataCube) SetTargetHost(req *http.Request) {
	req.URL.Scheme = "https"
	req.URL.Host = "api.consultasdeveiculos.com"
	req.Host = "api.consultasdeveiculos.com"
}

func (*DataCube) CacheResponse(key string, resp *http.Response) {
}

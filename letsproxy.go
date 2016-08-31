package letsproxy

import (
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

func Proxy(target string) func(w http.ResponseWriter, r *http.Request) {
	url, err := url.Parse("http://" + target)
	if err != nil {
		log.Fatal(err)
	}
	httpProxy := httputil.NewSingleHostReverseProxy(url)
	return func(w http.ResponseWriter, r *http.Request) {
		if !IsWebSocket(r) {
			httpProxy.ServeHTTP(w, r)
		} else {
			d, err := net.Dial("tcp", target)
			if err != nil {
				http.Error(w, "Error contacting backend server.", 500)
				log.Printf("Error dialing websocket backend %s: %v", target, err)
				return
			}
			hj, ok := w.(http.Hijacker)
			if !ok {
				http.Error(w, "Internal Error: Not Hijackable", 500)
				return
			}
			nc, _, err := hj.Hijack()
			if err != nil {
				log.Printf("Hijack error: %v", err)
				return
			}
			defer nc.Close()
			defer d.Close()

			// copy the request to the target first
			err = r.Write(d)
			if err != nil {
				log.Printf("Error copying request to target: %v", err)
				return
			}

			errc := make(chan error, 2)
			cp := func(dst io.Writer, src io.Reader) {
				_, err := io.Copy(dst, src)
				errc <- err
			}
			go cp(d, nc)
			go cp(nc, d)
			<-errc
		}
	}
}

func IsWebSocket(req *http.Request) bool {
	log.Println("Request: ", req.Proto, req.Host, req.RemoteAddr, req.Method, req.Cookies(), req.URL.String())

	conn_hdr := ""
	conn_hdrs := req.Header["Connection"]
	if len(conn_hdrs) > 0 {
		conn_hdr = conn_hdrs[0]
	}

	upgrade_websocket := false
	if strings.ToLower(conn_hdr) == "upgrade" {
		upgrade_hdrs := req.Header["Upgrade"]
		if len(upgrade_hdrs) > 0 {
			upgrade_websocket = (strings.ToLower(upgrade_hdrs[0]) == "websocket")
		}
	}

	return upgrade_websocket
}

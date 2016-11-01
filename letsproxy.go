package letsproxy

import (
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

func Proxy(target string) func(w http.ResponseWriter, r *http.Request) {
	var err error
	url, err := url.Parse("http://" + target)
	if err != nil {
		log.Fatal(err)
	}
	httpProxy := httputil.NewSingleHostReverseProxy(url)
	return func(w http.ResponseWriter, r *http.Request) {
		if !IsWebSocket(r) {
			//log.Println("Not WebSocket:", r.URL.String())
			httpProxy.ServeHTTP(w, r)
		} else {
			//log.Println("WebSocket:", r.URL.String())
			dialer := net.Dialer{KeepAlive: time.Second * 10}
			d, err := dialer.Dial("tcp", target)
			if err != nil {
				log.Printf("ERROR: dialing websocket backend %s: %v\n", target, err)
				http.Error(w, "Error contacting backend server.", 500)
				return
			}
			hj, ok := w.(http.Hijacker)
			if !ok {
				log.Println("ERROR: Not Hijackable")
				http.Error(w, "Internal Error: Not Hijackable", 500)
				return
				return
			}
			nc, _, err := hj.Hijack()
			if err != nil {
				log.Printf("ERROR: Hijack error: %v\n", err)
				return
			}
			defer nc.Close()
			defer d.Close()

			// copy the request to the target first
			err = r.Write(d)
			if err != nil {
				log.Printf("ERROR: copying request to target: %v\n", err)
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
	//log.Println("IsWebSocket called: ", req.URL.String())
	//log.Println("Connection", req.Header["Connection"])
	//log.Println("Upgrade:", req.Header["Upgrade"])

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

// Logging
type LogRecord struct {
	http.ResponseWriter
	status int
}

func (r *LogRecord) Write(p []byte) (int, error) {
	return r.ResponseWriter.Write(p)
}

func (r *LogRecord) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func WrapHandler(f http.Handler, verbose bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		record := &LogRecord{
			ResponseWriter: w,
		}
		f.ServeHTTP(record, r)
		if record.status == http.StatusBadRequest || verbose {
			log.Println(r.RemoteAddr, record.status, r.Method, r.Host, r.URL.Path)
		}
	}
}

package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/abbot/go-http-auth"
	"github.com/varung/letsproxy"
	"rsc.io/letsencrypt"
)

func main() {
	target := flag.String("target", "127.0.0.1:8888", "where to proxy the connection to")
	flag.Parse()
	log.Println("target: ", *target)
	proxyFunc := letsproxy.Proxy(*target)
	secrets := auth.HtpasswdFileProvider("example.htpasswd")
	authenticator := auth.NewBasicAuthenticator("varunlab.com", secrets)
	http.HandleFunc("/", authenticator.Wrap(func(w http.ResponseWriter, r *auth.AuthenticatedRequest) {
		proxyFunc(w, &(r.Request))
	}))
	var m letsencrypt.Manager
	if err := m.CacheFile("letsencrypt.cache"); err != nil {
		log.Fatal(err)
	}
	log.Fatal(m.Serve())
}

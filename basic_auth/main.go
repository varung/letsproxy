package main

import (
	"log"
	"net/http"
	"sync"

	"github.com/abbot/go-http-auth"
	"github.com/varung/letsproxy"
	"rsc.io/letsencrypt"
)

var mut = sync.Mutex{}

func main() {
	proxyFunc := letsproxy.Proxy("127.0.0.1:8888")
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

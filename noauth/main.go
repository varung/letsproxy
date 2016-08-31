package main

import (
	"log"
	"net/http"
	"sync"

	"github.com/varung/letsproxy"
	"rsc.io/letsencrypt"
)

var mut = sync.Mutex{}

func main() {
	proxy := letsproxy.Proxy("127.0.0.1:8888")
	http.HandleFunc("/", proxy)
	var m letsencrypt.Manager
	if err := m.CacheFile("letsencrypt.cache"); err != nil {
		log.Fatal(err)
	}
	log.Fatal(m.Serve())
}

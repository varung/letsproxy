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

	// TODO: change to flag
	var files_root string = "/"
	fileserver := http.FileServer(http.Dir(files_root))

	http.HandleFunc("/files/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("cache-control", "no-cache")
		fileserver.ServeHTTP(w, r)
	})

	http.HandleFunc("/check", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
		w.WriteHeader(200)
	})

	// equivalent nginx: try_files $uri $uri/ $uri/index.html @compute;
	proxy := http.HandlerFunc(letsproxy.Proxy("127.0.0.1:8282"))
	wrapped := letsproxy.WrapHandler(proxy, true)
	var public_root string = "./public"
	public_dir := http.Dir(public_root)
	public_fs := http.FileServer(public_dir)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// if file not present in public directory, then proxy
		f, _ := public_dir.Open(r.URL.Path)
		if f == nil {
			wrapped.ServeHTTP(w, r)
		} else {
			f.Close()
			public_fs.ServeHTTP(w, r)
		}
	})

	// TODO:
	// /exec
	// /reboot
	// /upload

	// letsencrypt specific stuff
	var m letsencrypt.Manager
	if err := m.CacheFile("letsencrypt.cache"); err != nil {
		log.Fatal(err)
	}
	log.Fatal(m.Serve())
}

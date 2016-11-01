package main

import (
	"bytes"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"encoding/json"

	"github.com/varung/letsproxy"
	"rsc.io/letsencrypt"
)

var mut = sync.Mutex{}

func check(err error) {
	if err != nil {
		log.Println("Error:", err)
	}
}

type ExecResult struct {
	Error  int    `json:"error"`
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
	Cmd    string `json:"cmd"`
	Debug  string `json:"debug"`
}

func main() {

	log.SetFlags(log.Lshortfile | log.LUTC)
	// TODO: change to flag
	var files_root string = "/"
	fileserver := http.FileServer(http.Dir(files_root))
	http.Handle("/files/", http.StripPrefix("/files/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("cache-control", "no-cache")
		fileserver.ServeHTTP(w, r)
	})))

	http.HandleFunc("/check", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
		w.WriteHeader(200)
	})

	http.HandleFunc("/exec", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dec := json.NewDecoder(r.Body)
		var dat map[string]string
		err := dec.Decode(&dat)
		if err != nil {
			log.Println("ERROR: /exec", err)
			w.WriteHeader(500)
			return
		}
		cmd := dat["cmd"]
		log.Println(cmd)

		doneChan := make(chan ExecResult)
		bash := exec.Command("bash")
		go func() {
			bash.Stdin = strings.NewReader(cmd)
			var stdout, stderr bytes.Buffer
			bash.Stdout = &stdout
			bash.Stderr = &stderr

			// TODO: kill it automatically after some amount of time?
			result := ExecResult{Cmd: cmd}
			if err := bash.Run(); err != nil {
				if exiterr, ok := err.(*exec.ExitError); ok {
					// The program has exited with an exit code != 0
					// This works on both Unix and Windows. Although package
					// syscall is generally platform dependent, WaitStatus is
					// defined for both Unix and Windows and in both cases has
					// an ExitStatus() method with the same signature.
					if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
						log.Printf("Exit Status: %d", status.ExitStatus())
						result.Error = status.ExitStatus()
					}
				} else {
					log.Println(err)
					result.Error = -1
					result.Debug = "Error calling Run(): " + err.Error()
				}
			}
			result.Stdout = stdout.String()
			result.Stderr = stderr.String()
			doneChan <- result
		}()

		final := ExecResult{}
		select {
		case final = <-doneChan:
			log.Println("got result")
		case <-time.After(5 * time.Second):
			bash.Process.Kill()
			final.Debug = "Timed Out"
		}

		enc := json.NewEncoder(w)
		err = enc.Encode(final)
		if err != nil {
			log.Println(err)
		}
	}))

	// equivalent nginx: try_files $uri $uri/ $uri/index.html @compute;
	proxy := http.HandlerFunc(letsproxy.Proxy("127.0.0.1:8282"))
	var public_root string = "/opt/web-terminal/public"
	public_dir := http.Dir(public_root)
	public_fs := http.FileServer(public_dir)
	http.HandleFunc("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("\n\ntryfiles: ", r.URL.Path)
		// if file not present in public directory, then proxy
		f, _ := public_dir.Open(r.URL.Path)
		if f == nil {
			log.Println("proxying", r.URL.Path)
			proxy.ServeHTTP(w, r)
		} else {
			log.Println("serving", r.URL.Path)
			f.Close()
			public_fs.ServeHTTP(w, r)
		}
	}))

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

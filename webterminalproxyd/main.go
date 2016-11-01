package main

import (
	"bytes"
	"fmt"
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

func HandleCheck(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
	w.WriteHeader(200)
}

type ExecResult struct {
	Error  int    `json:"error"` // probably should really be called status
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
	Cmd    string `json:"cmd"`
	Debug  string `json:"debug"`
}

func ExecHelper(w http.ResponseWriter, cmd string) ExecResult {
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
		log.Println("Exec Process Returned")
	case <-time.After(15 * time.Second): // TODO: take as argument
		log.Println("Exec Process Timed Out. Killing.")
		bash.Process.Kill()
		final.Debug = "Timed Out"
		final.Error = -1
	}
	return final
}

func HandleExec(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)
	var dat map[string]string
	err := dec.Decode(&dat)
	if err != nil {
		log.Println("ERROR: /exec", err)
		w.WriteHeader(500)
		return
	}
	cmd := dat["cmd"]
	final := ExecHelper(w, cmd)
	enc := json.NewEncoder(w)
	err = enc.Encode(final)
	if err != nil {
		log.Println(err)
	}
}

func HandleReboot(w http.ResponseWriter, r *http.Request) {
	log.Println("HandleReboot")
	result := ExecHelper(w, "/opt/web-terminal/reboot.sh")
	w.WriteHeader(200)
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "%d\n%s\n%s", result.Error, result.Stdout, result.Stderr)
}

func main() {

	log.SetFlags(log.Lshortfile | log.LUTC)

	// TODO: change following to flags
	files_root := "/"
	fileserver := http.FileServer(http.Dir(files_root))
	public_root := "/opt/web-terminal/public"
	public_dir := http.Dir(public_root)
	public_fs := http.FileServer(public_dir)

	// handle everything
	http.Handle("/files/", http.StripPrefix("/files/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("cache-control", "no-cache")
		fileserver.ServeHTTP(w, r)
	})))
	http.HandleFunc("/check", HandleCheck)
	http.HandleFunc("/exec", HandleExec)
	http.HandleFunc("/reboot", HandleReboot)

	// handle uploads
	uploader := letsproxy.Uploader{TmpDir: "/tmp"}
	http.Handle("/upload", &uploader)

	// equivalent nginx: try_files $uri $uri/ $uri/index.html @compute;
	proxy := http.HandlerFunc(letsproxy.Proxy("127.0.0.1:8282"))
	http.HandleFunc("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//log.Println("\n\ntryfiles: ", r.URL.Path)
		// if file not present in public directory, then proxy
		f, _ := public_dir.Open(r.URL.Path)
		if f == nil {
			//log.Println("proxying", r.URL.Path)
			proxy.ServeHTTP(w, r)
		} else {
			//log.Println("serving", r.URL.Path)
			f.Close()
			public_fs.ServeHTTP(w, r)
		}
	}))

	// letsencrypt specific stuff
	var m letsencrypt.Manager
	if err := m.CacheFile("letsencrypt.cache"); err != nil {
		log.Fatal(err)
	}
	log.Fatal(m.Serve())
}

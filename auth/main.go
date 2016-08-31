package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/apexskier/httpauth"
	"github.com/varung/letsproxy"
	"rsc.io/letsencrypt"
)

var (
	backend     httpauth.AuthBackend
	aaa         httpauth.Authorizer
	roles       map[string]httpauth.Role
	port        = 8009
	backendfile = "auth.ldb"
	mut         = sync.Mutex{}
)

func main() {

	var err error
	os.Mkdir(backendfile, 0755)
	// create the backend
	backend, err = httpauth.NewLeveldbAuthBackend(backendfile)
	if err != nil {
		panic(err)
	}

	roles = make(map[string]httpauth.Role)
	roles["admin"] = 80
	aaa, err = httpauth.NewAuthorizer(backend, []byte("cookie-encryption-key"), "user", roles)

	// create a default user if not already there
	username := "admin"
	_, e := backend.User(username)
	if e == httpauth.ErrMissingUser {
		defaultUser := httpauth.UserData{Username: username, Role: "admin"}
		err = backend.SaveUser(defaultUser)
		if err != nil {
			panic(err)
		}
		// Update user with a password and email address
		err = aaa.Update(nil, nil, username, "somecrazy2", "admin@localhost.com")
		if err != nil {
			panic(err)
		}
	}

	proxy := letsproxy.Proxy("127.0.0.1:8888")

	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			getLogin(w, r)
		case "POST":
			postLogin(w, r)
		}
	})
	http.HandleFunc("/admin", handleAdmin)
	http.HandleFunc("/info", handlePage)
	http.HandleFunc("/logout", handleLogout)
	//r.HandleFunc(h/add_user", postAddUser).Methods("POST")
	//r.HandleFunc("/change", postChange).Methods("POST")
	//r.HandleFunc("/", handlePage).Methods("GET")
	http.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		if err := aaa.Authorize(rw, req, true); err != nil {
			fmt.Println(err)
			http.Redirect(rw, req, "/login", http.StatusSeeOther)
			return
		}
		proxy(rw, req)
		return
	})

	var m letsencrypt.Manager
	if err := m.CacheFile("letsencrypt.cache"); err != nil {
		log.Fatal(err)
	}
	log.Fatal(m.Serve())
}

func getLogin(rw http.ResponseWriter, req *http.Request) {
	messages := aaa.Messages(rw, req)
	log.Println("Login")
	fmt.Fprintf(rw, `
        <html>
        <head><title>Login</title></head>
        <body>
        <h1>Httpauth example</h1>
        <h2>Entry Page</h2>
        <p><b>Messages: %v</b></p>
        <h3>Login</h3>
        <form action="/login" method="post" id="login">
            <input type="text" name="username" placeholder="username"><br>
            <input type="password" name="password" placeholder="password"></br>
            <button type="submit">Login</button>
        </form>
        <!--<h3>Register</h3>
        <form action="/register" method="post" id="register">
            <input type="text" name="username" placeholder="username"><br>
            <input type="password" name="password" placeholder="password"></br>
            <input type="email" name="email" placeholder="email@example.com"></br>
            <button type="submit">Register</button>
        </form>-->
        </body>
        </html>
        `, messages)
}

func handlePage(rw http.ResponseWriter, req *http.Request) {
	if err := aaa.Authorize(rw, req, true); err != nil {
		fmt.Println(err)
		http.Redirect(rw, req, "/login", http.StatusSeeOther)
		return
	}
	if user, err := aaa.CurrentUser(rw, req); err == nil {
		type data struct {
			User httpauth.UserData
		}
		d := data{User: user}
		t, err := template.New("page").Parse(`
            <html>
            <head><title>Secret page</title></head>
            <body>
                <h1>Httpauth example<h1>
                {{ with .User }}
                    <h2>Hello {{ .Username }}</h2>
                    <p>Your role is '{{ .Role }}'. Your email is {{ .Email }}.</p>
                    <p>{{ if .Role | eq "admin" }}<a href="/admin">Admin page</a> {{ end }}<a href="/logout">Logout</a></p>
                {{ end }}
                <form action="/change" method="post" id="change">
                    <h3>Change email</h3>
                    <p><input type="email" name="new_email" placeholder="new email"></p>
                    <button type="submit">Submit</button>
                </form>
            </body>
            `)
		if err != nil {
			panic(err)
		}
		t.Execute(rw, d)
	}
}

func postLogin(rw http.ResponseWriter, req *http.Request) {
	username := req.PostFormValue("username")
	password := req.PostFormValue("password")
	if err := aaa.Login(rw, req, username, password, "/"); err == nil ||
		(err != nil && strings.Contains(err.Error(), "already authenticated")) {
		log.Println("Redirecting user to /")
		http.Redirect(rw, req, "/", 301)
	} else if err != nil {
		fmt.Println(err)
		http.Redirect(rw, req, "/login", http.StatusSeeOther)
	}
}
func postChange(rw http.ResponseWriter, req *http.Request) {
	email := req.PostFormValue("new_email")
	aaa.Update(rw, req, "", "", email)
	log.Println("Redirecting user to /")
	http.Redirect(rw, req, "/", http.StatusSeeOther)
}
func handleAdmin(rw http.ResponseWriter, req *http.Request) {
	if err := aaa.AuthorizeRole(rw, req, "admin", true); err != nil {
		fmt.Println(err)
		http.Redirect(rw, req, "/login", http.StatusSeeOther)
		return
	}
	if user, err := aaa.CurrentUser(rw, req); err == nil {
		type data struct {
			User  httpauth.UserData
			Roles map[string]httpauth.Role
			Users []httpauth.UserData
			Msg   []string
		}
		messages := aaa.Messages(rw, req)
		users, err := backend.Users()
		if err != nil {
			panic(err)
		}
		d := data{User: user, Roles: roles, Users: users, Msg: messages}
		t, err := template.New("admin").Parse(`
            <html>
            <head><title>Admin page</title></head>
            <body>
                <h1>Httpauth example<h1>
                <h2>Admin Page</h2>
                <p>{{.Msg}}</p>
                {{ with .User }}<p>Hello {{ .Username }}, your role is '{{ .Role }}'. Your email is {{ .Email }}.</p>{{ end }}
                <p><a href="/">Back</a> <a href="/logout">Logout</a></p>
                <h3>Users</h3>
                <ul>{{ range .Users }}<li>{{.Username}}</li>{{ end }}</ul>
                <form action="/add_user" method="post" id="add_user">
                    <h3>Add user</h3>
                    <p><input type="text" name="username" placeholder="username"><br>
                    <input type="password" name="password" placeholder="password"><br>
                    <input type="email" name="email" placeholder="email"><br>
                    <select name="role">
                        <option value="">role<option>
                        {{ range $key, $val := .Roles }}<option value="{{$key}}">{{$key}} - {{$val}}</option>{{ end }}
                    </select></p>
                    <button type="submit">Submit</button>
                </form>
            </body>
            `)
		if err != nil {
			panic(err)
		}
		t.Execute(rw, d)
	}
}
func handleLogout(rw http.ResponseWriter, req *http.Request) {
	if err := aaa.Logout(rw, req); err != nil {
		fmt.Println(err)
		// this shouldn't happen
		return
	}
	log.Println("Redirecting to /")
	http.Redirect(rw, req, "/", http.StatusSeeOther)
}

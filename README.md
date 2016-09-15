# letsproxy
golang https and websocket reverse proxy using letsencrypt to automatically setup SSL certs
authenticated versions as well. Websockets work fine.

Why? You want to run a web server behind SSL and authentication. letsproxy uses let's encrypt to automatically generate a SSL certificate, and then proxies traffic to your server. Optionally, it can protect access via http basic auth, or session based auth.
I use this to access my ipython notebook instance running on a server more securely.


# Usage:

```
go get -u github.com/varung/letsproxy
# cd to the folder
go build
cd basic_auth
sudo ./basic_auth --target {IP}:{PORT}
# e.g., sudo ./basic_auth --target 127.0.0.1:8080
# user,pass: test,test
```

since letsproxy wants to listen on 443, you'll need to run it with sudo, or, use the included bash script to give letsproxy the privileged port capability

basic_auth uses http basic auth
auth uses a very simple system for managing accounts and passwords with sessions
noauth, does no auth (i.e., still ssl, but no password required)

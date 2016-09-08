# letsproxy
golang https and websocket reverse proxy using letsencrypt to automatically setup SSL certs
authenticated versions as well

Why? You want to run a web server behind SSL and authentication. letsproxy uses let's encrypt to automatically generate a SSL certificate, and then proxies traffic to your server. Optionally, it can protect access via http basic auth, or session based auth.
I use this to access my ipython notebook instance running on a server more securely.

# Usage:

```
go get -u github.com/varung/letsproxy
# cd to the folder
go build
sudo ./letsproxy --target {IP}:{PORT}
```

since letsproxy wants to listen on 443, you'll need to run it with sudo, or, use the included bash script to give letsproxy the privileged port capability

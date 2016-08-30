# letsproxy
golang https and websocket reverse proxy using letsencrypt to automatically setup SSL certs

Why?
iPython notebook is a http/websockets app that does not have any security measures in place to protect access from the WAN/public internet.

This simple proxy is intended to listen on public ports, authenticate access if user is not logged in, and then proxy all traffic to ipython notebook.

Using letsencrypt, this proxy automatically generates a SSL cert as well, avoiding the need for self-signed certs and browser warnings

As a user, you need to have a domain name and assign it to your server. After that this will take care of the rest.

It can of course be used in front of any web app, not just ipython notebook, but that was my motivating example.

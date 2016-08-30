# letsproxy
golang https and websocket reverse proxy using letsencrypt to automatically setup SSL certs

Why?
iPython notebook is a http/websockets app that does not have any security measures in place for accessing over a remote network

This simple proxy is intended to listen on public ports, authenticate access if user is not logged in, and then proxy all traffic to ipython notebook.

Using letsencrypt, this proxy automatically generates a SSL cert as well, avoiding the need for self-signed certs and browser warnings

As a user, you need to have a domain name and assign it to your server. After that this will take care of the rest.

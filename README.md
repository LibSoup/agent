# LibSoup system agent
A simple Go app that can run on virtually any modern *nix system.

This agent gathers a minimal amount of data and uploads it to libsoup.com.  Gathered data consists of:
  * hostname
  * libraries in use (with # of pids using it)
  * operating system name

That's it!  Plain and simple.  No telemetry, no unnecessary datapoints etc.

See https://libsoup.com/agent for more information.


### Installing
```
go get github.com/LibSoup/libsoup-agent
```

### Building
```
cd ~/go/src
git clone git@github.com:LibSoup/libsoup-agent.git
cd libsoup-agent
go build .
```
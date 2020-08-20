# Cacheserver

Cacheserver is a very simply http caching proxy.  
This project is meant to be a proxy for (large) downloads on http servers and not meant to proxy a human interfaceing website.

# Install

## Go get

```sh
go get -u github.com/chrisvdg/cacheserver
```

## Build from this repository
```sh
go build

# install in $GO/bin
go install
```

# Usage
```sh
# Listen to localhost only on port 8000 and proxy google
cacheserver -l "127.0.0.1:8000" -p http://google.com

# Listen to all incoming requests on port 9000 and proxy download.archive and show debug output
cacheserver -l ":9000" -p http://download.archive -v
```


# Docker

A docker file has been added to generate a docker image

To build the docker image
```sh
docker build -t cacheserver .
```

Run a container with image
```sh
docker run --rm -ti cacheserver -l ":80" -p http://download.archive -v
```

A prebuilt image has been uploaded to docker hub.
To import the image
```sh
docker pull chrisvdg/cacheserver
```

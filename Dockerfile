
ARG go_version=latest
ARG alpine_version=latest


FROM golang:${go_version} as builder

WORKDIR /go/src/cacheserver

COPY . .

RUN go get -d -v ./...
RUN CGO_ENABLED=0 go install -v ./...


FROM alpine:$alpine_version
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
RUN apk add tzdata && cp /usr/share/zoneinfo/Europe/Brussels /etc/localtime && echo "Europe/Brussels" >  /etc/timezone && apk del tzdata

COPY --from=builder /go/bin/cacheserver /bin

ENTRYPOINT [ "cacheserver" ]
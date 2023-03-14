# syntax=docker/dockerfile:1
# https://docs.docker.com/language/golang/build-images/

FROM alpine:edge
RUN apk add --no-cache --update go gcc g++
COPY / ./
RUN CGO_ENABLED=1 GOOS=linux go build -o /adr
RUN apk add --no-cache --update caddy
RUN printf "*:8080\nencode zstd gzip\n\nreverse_proxy localhost:9090" | tee -a /etc/caddy/Caddyfile
RUN caddy start /etc/caddy/Caddyfile
EXPOSE 8080
CMD ["/adr"]
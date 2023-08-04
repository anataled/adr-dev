# syntax=docker/dockerfile:1
# https://docs.docker.com/language/golang/build-images/

FROM alpine:3.18
RUN apk add --no-cache --update go gcc g++
COPY / ./
RUN CGO_ENABLED=1 GOOS=linux go build -o /adr
EXPOSE 8080
CMD ["/adr"]
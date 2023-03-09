# syntax=docker/dockerfile:1
# https://docs.docker.com/language/golang/build-images/

FROM alpine:edge AS build
RUN apk add --no-cache --update go gcc g++
COPY / ./
RUN CGO_ENABLED=1 GOOS=linux go build -o /adr

FROM alpine:edge
COPY --from=build /adr /adr
EXPOSE 8080
CMD ["/adr"]
FROM golang:1.18 AS build

ADD . /api
WORKDIR /api
RUN go build -o bin/main cmd/main.go

FROM ubuntu:20.04

WORKDIR /usr/src/app

# COPY . .
COPY --from=build /api/bin/main .
COPY --from=build /api/config.json .

EXPOSE 3001
CMD ./main

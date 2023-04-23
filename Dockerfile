FROM golang:1.20.2-alpine AS build

WORKDIR /app
COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY *.go ./
RUN go build -o /goapp

FROM alpine
WORKDIR /app
COPY --from=build /goapp /app
ENTRYPOINT ./goapp
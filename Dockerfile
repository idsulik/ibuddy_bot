FROM golang:1.20.2-alpine AS build

WORKDIR /app
COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY *.go ./
COPY database/*.go ./database/
RUN go build -o /goapp

FROM alpine
RUN apk add  --no-cache ffmpeg
WORKDIR /app
COPY --from=build /goapp /app
ENTRYPOINT ./goapp
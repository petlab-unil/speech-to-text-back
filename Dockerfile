FROM golang:1.14-alpine AS build_base

WORKDIR /app

COPY src src
COPY go.mod go.mod
COPY go.sum go.sum

RUN go mod download
RUN go build -o ./out/speech-to-text-back ./src/main.go

FROM alpine:3.9
RUN apk add ca-certificates
WORKDIR	/app
COPY --from=build_base /app/out/speech-to-text-back /app/speech-to-text-back
COPY unil.json /app/unil.json

EXPOSE 8080

CMD ["/app/speech-to-text-back"]

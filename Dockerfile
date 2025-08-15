FROM golang:1.24.5-alpine AS builder

LABEL org.opencontainers.image.source=https://github.com/DeepSpace2/plugnpin

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o auto_pi_hole_npm main.go

FROM alpine:3.22.1

RUN apk add --no-cache tzdata

RUN mkdir /root/.docker && echo "{}" > /root/.docker/config.json

WORKDIR /app

COPY --from=builder /app/auto_pi_hole_npm .

CMD [ "./auto_pi_hole_npm" ]


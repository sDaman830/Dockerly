FROM golang:1.21-bullseye as builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o dockerly main.go rootfs.go cgroup.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/dockerly /app/dockerly
ENTRYPOINT ["/app/dockerly"]

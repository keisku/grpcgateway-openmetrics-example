FROM golang:1.17 as builder
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64
WORKDIR /workspace
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download
COPY . .
RUN go build -a -o /app main.go

FROM gcr.io/distroless/static:nonroot
USER nonroot:nonroot
COPY --from=builder /app /app
ENTRYPOINT ["/app"]

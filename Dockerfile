FROM golang:alpine
WORKDIR /app
COPY ./ ./
RUN go build -o project ./cmd/main.go
CMD ["./project"]
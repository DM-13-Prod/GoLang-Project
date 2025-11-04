package main

import (
	"log"
	"net"
	"path/filepath"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"

	"todo/internal/grpcapi"
	"todo/internal/grpcserver"
	"todo/internal/repository"
	"todo/internal/service"
)

func main() {
	_ = godotenv.Load()

	// Всегда смотрим относительно корня проекта
	dataPath := filepath.Join("cmd", "data", "tasks.json")

	log.Println("[DATA PATH]:", dataPath)

	store := repository.NewJSONStore("cmd/data/tasks.json")
	svc, err := service.New(store)
	if err != nil {
		log.Fatalf("service init error: %v", err)
	}

	addr := "127.0.0.1:50505"
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen %s: %v", addr, err)
	}

	s := grpc.NewServer()
	grpcapi.RegisterTodoServiceServer(s, grpcserver.New(svc))

	log.Println("[gRPC] listening on", addr)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("serve error: %v", err)
	}
}
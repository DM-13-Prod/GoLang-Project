package web

import (
	"todo/internal/service"
	"fmt"
	"net/http"

	"github.com/swaggo/http-swagger"
)

type Server struct {
	svc service.TaskUseCase
}

func New(uc service.TaskUseCase) *Server {
	return &Server{svc: uc}
}

func (s *Server) Start(port int) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/login", s.handleLogin)
	mux.HandleFunc("/api/item", s.handleCreateItem)        // POST
	mux.HandleFunc("/api/items", s.handleListItems)        // GET all
	mux.HandleFunc("/api/item/", s.handleItemByID)         // GET, PUT, DELETE (/api/item/{id})
	
	mux.Handle("/swagger/", httpSwagger.WrapHandler)

	addr := fmt.Sprintf(":%d", port)
	fmt.Println("[Web] Веб сервер стартовал на ", addr)
	return http.ListenAndServe(addr, mux)
}
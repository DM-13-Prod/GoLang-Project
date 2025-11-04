package grpcserver

import (
	"context"
	"todo/internal/grpcapi"
	"todo/internal/model"
	"todo/internal/service"
	"time"
)

type Server struct {
	grpcapi.UnimplementedTodoServiceServer
	svc service.TaskUseCase
}

func New(svc service.TaskUseCase) *Server {
	return &Server{svc: svc}
}

func (s *Server) Create(ctx context.Context, req *grpcapi.CreateTaskRequest) (*grpcapi.CreateTaskResponse, error) {
	var due *time.Time
	if req.DueAt != "" {
		t, err := time.Parse("2006-01-02", req.DueAt)
		if err == nil {
			due = &t
		}
	}
	id, err := s.svc.Add(req.Title, req.Description, model.Priority(req.Priority), due)
	if err != nil {
		return nil, err
	}
	return &grpcapi.CreateTaskResponse{Id: int64(id)}, nil
}

func (s *Server) Update(ctx context.Context, req *grpcapi.UpdateTaskRequest) (*grpcapi.Task, error) {
	id := model.ID(req.Id)
	if req.Title != "" {
		_ = s.svc.UpdateTitle(id, req.Title)
	}
	if req.Description != "" {
		_ = s.svc.UpdateDesc(id, req.Description)
	}
	if req.Status != "" {
		_ = s.svc.SetStatus(id, model.Status(req.Status))
	}
	if req.Priority > 0 {
		_ = s.svc.SetPriority(id, model.Priority(req.Priority))
	}
	if req.DueAt != "" {
		if req.DueAt == "-" {
			_ = s.svc.ClearDue(id)
		} else if t, err := time.Parse("2006-01-02", req.DueAt); err == nil {
			_ = s.svc.SetDue(id, t)
		}
	}
	for _, t := range s.svc.List(nil) {
		if t.ID() == id {
			return dtoToProto(t), nil
		}
	}
	return nil, nil
}

func (s *Server) Delete(ctx context.Context, req *grpcapi.TaskID) (*grpcapi.Empty, error) {
	_ = s.svc.Delete(model.ID(req.Id))
	return &grpcapi.Empty{}, nil
}

func (s *Server) Get(ctx context.Context, req *grpcapi.TaskID) (*grpcapi.Task, error) {
	for _, t := range s.svc.List(nil) {
		if t.ID() == model.ID(req.Id) {
			return dtoToProto(t), nil
		}
	}
	return nil, nil
}

func (s *Server) List(ctx context.Context, _ *grpcapi.Empty) (*grpcapi.TaskList, error) {
	list := s.svc.List(nil)
	resp := &grpcapi.TaskList{}
	for _, t := range list {
		resp.Items = append(resp.Items, dtoToProto(t)) // было *dtoToProto(t)
	}
	return resp, nil
}

func dtoToProto(t *model.Task) *grpcapi.Task {
	var due, comp string
	if t.DueAt() != nil {
		due = t.DueAt().Format("2006-01-02")
	}
	if t.CompletedAt() != nil {
		comp = t.CompletedAt().Format("2006-01-02 15:04")
	}
	return &grpcapi.Task{
		Id:          int64(t.ID()),
		Title:       t.Title(),
		Description: t.Description(),
		Status:      string(t.Status()),
		Priority:    int32(t.Priority()),
		DueAt:       due,
		CreatedAt:   t.CreatedAt().Format("2006-01-02 15:04"),
		UpdatedAt:   t.UpdatedAt().Format("2006-01-02 15:04"),
		CompletedAt: comp,
	}
}
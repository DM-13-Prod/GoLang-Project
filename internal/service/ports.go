package service

import (
	"context"
	"time"

	"todo/internal/model"
)

// Store — абстракция хранилища для задач.
// Специально оставляем DTO, чтобы адаптер JSON был тонким.
type Store interface {
	Load() ([]model.TaskDTO, error)
	Save([]model.TaskDTO) error
}

// TaskUseCase — контракт бизнес-логики для веба/гRPC.
type TaskUseCase interface {
	Add(title, desc string, p model.Priority, due *time.Time) (model.ID, error)
	RenumberIDs() error
	List(filter *model.Status) []*model.Task
	UpdateTitle(id model.ID, title string) error
	UpdateDesc(id model.ID, desc string) error
	SetStatus(id model.ID, st model.Status) error
	SetPriority(id model.ID, p model.Priority) error
	SetDue(id model.ID, due time.Time) error
	ClearDue(id model.ID) error
	Delete(id model.ID) error
}

// Событие аудита для Redis
type Event struct {
	Op     string         `json:"op"`
	TaskID model.ID       `json:"task_id,omitempty"`
	At     time.Time      `json:"at"`
	Before *model.TaskDTO `json:"before,omitempty"`
	After  *model.TaskDTO `json:"after,omitempty"`
}

type AuditLogger interface {
	LogEvent(ctx context.Context, e Event) error
}

// Глобально настраиваемый логгер (опционально)
var Logger AuditLogger
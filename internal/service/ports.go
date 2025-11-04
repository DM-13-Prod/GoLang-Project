package service

import (
	"time" 
	"todo/internal/model"
)
// Специально оставляем DTO, чтобы адаптер JSON был тонким.
type Store interface {
	Load() ([]model.TaskDTO, error)
	Save([]model.TaskDTO) error
}

// TaskUseCase — контракт бизнес-логики для веба/гРПС.
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
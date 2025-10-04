package model

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type ID int64

type Status string

const (
	StatusNew        Status = "new"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
	StatusPaused     Status = "paused"
	StatusCanceled   Status = "canceled"
)

func (s Status) Valid() bool {
	switch s {
	case StatusNew, StatusInProgress, StatusDone, StatusPaused, StatusCanceled:
		return true
	default:
		return false
	}
}

type Priority int

const (
	PriorityLow Priority = iota + 1
	PriorityMedium
	PriorityHigh
)

func (p Priority) Valid() bool {
	return p >= PriorityLow && p <= PriorityHigh
}

// Встраиваемая мета (пример встраивания и приватных полей)
type meta struct {
	createdAt   time.Time
	updatedAt   time.Time
	completedAt *time.Time
}

func (m *meta) touch() {
	m.updatedAt = time.Now()
}

type Task struct {
	meta
	id          ID
	title       string
	description string
	status      Status
	priority    Priority
	dueAt       *time.Time
}

func NewTask(title, description string) (*Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, errors.New("title is empty")
	}
	now := time.Now()
	return &Task{
		id:          ID(now.UnixNano()),
		title:       title,
		description: strings.TrimSpace(description),
		status:      StatusNew,
		priority:    PriorityMedium,
		meta: meta{
			createdAt: now,
			updatedAt: now,
		},
	}, nil
}

// Геттеры
func (t *Task) ID() ID                   { return t.id }
func (t *Task) Title() string            { return t.title }
func (t *Task) Description() string      { return t.description }
func (t *Task) Status() Status           { return t.status }
func (t *Task) Priority() Priority       { return t.priority }
func (t *Task) DueAt() *time.Time        { return t.dueAt }
func (t *Task) CreatedAt() time.Time     { return t.createdAt }
func (t *Task) UpdatedAt() time.Time     { return t.updatedAt }
func (t *Task) CompletedAt() *time.Time  { return t.completedAt }

// Методы изменения (валидация)
func (t *Task) SetTitle(title string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return errors.New("title is empty")
	}
	t.title = title
	t.touch()
	return nil
}

func (t *Task) SetDescription(desc string) {
	t.description = strings.TrimSpace(desc)
	t.touch()
}

func (t *Task) SetStatus(s Status) error {
	if !s.Valid() {
		return fmt.Errorf("invalid status: %s", s)
	}
	t.status = s
	if s == StatusDone {
		now := time.Now()
		t.completedAt = &now
	} else {
		t.completedAt = nil
	}
	t.touch()
	return nil
}

func (t *Task) SetPriority(p Priority) error {
	if !p.Valid() {
		return fmt.Errorf("invalid priority: %d", p)
	}
	t.priority = p
	t.touch()
	return nil
}

func (t *Task) SetDueAt(d time.Time) {
	dd := d
	t.dueAt = &dd
	t.touch()
}

func (t *Task) ClearDue() {
	t.dueAt = nil
	t.touch()
}

// DTO для JSON-хранилища
type TaskDTO struct {
	ID          ID         `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	Status      Status     `json:"status"`
	Priority    Priority   `json:"priority"`
	DueAt       *time.Time `json:"due_at,omitempty"`

	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

func (t *Task) ToDTO() TaskDTO {
	return TaskDTO{
		ID:          t.id,
		Title:       t.title,
		Description: t.description,
		Status:      t.status,
		Priority:    t.priority,
		DueAt:       t.dueAt,
		CreatedAt:   t.createdAt,
		UpdatedAt:   t.updatedAt,
		CompletedAt: t.completedAt,
	}
}

func FromDTO(r TaskDTO) (*Task, error) {
	if strings.TrimSpace(r.Title) == "" {
		return nil, errors.New("record: empty title")
	}
	if !r.Status.Valid() {
		return nil, fmt.Errorf("record: bad status %q", r.Status)
	}
	if !r.Priority.Valid() {
		return nil, fmt.Errorf("record: bad priority %d", r.Priority)
	}
	return &Task{
		id:          r.ID,
		title:       strings.TrimSpace(r.Title),
		description: strings.TrimSpace(r.Description),
		status:      r.Status,
		priority:    r.Priority,
		dueAt:       r.DueAt,
		meta: meta{
			createdAt:   r.CreatedAt,
			updatedAt:   r.UpdatedAt,
			completedAt: r.CompletedAt,
		},
	}, nil
}
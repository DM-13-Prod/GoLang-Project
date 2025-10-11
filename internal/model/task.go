package model

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// ID — уникальный номер задачи, просто число
type ID int64

// Status — состояние задачи, с типом строки
type Status string

// Разные состояния, базовая реализация статусов
const (
	StatusNew        Status = "new"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
	StatusPaused     Status = "paused"
	StatusCanceled   Status = "canceled"
)

// Проверяет что статус нормальный, без левых значений
func (s Status) Valid() bool {
	switch s {
	case StatusNew, StatusInProgress, StatusDone, StatusPaused, StatusCanceled:
		return true
	default:
		return false
	}
}

// init — запускается один раз, просто сидает рандом
func init() {
	rand.Seed(time.Now().UnixNano())
}

// генерит случайный id для задачи, чтобы отличать друг от друга
func generateID() ID {
	return ID(rand.Int63n(1_000_000_000))
}

// Priority — важность задачи, типа низкий, средний и высокий
type Priority int

const (
	PriorityLow Priority = iota + 1
	PriorityMedium
	PriorityHigh
)

// Проверка приоритета, смотрит и проверяет
func (p Priority) Valid() bool {
	return p >= PriorityLow && p <= PriorityHigh
}

// meta — просто технич поля про время создания/обновления/завершения
type meta struct {
	createdAt   time.Time
	updatedAt   time.Time
	completedAt *time.Time
}

func (m *meta) touch() {
	m.updatedAt = time.Now()
}

// Task — основная структура задачи
type Task struct {
	meta
	id          ID
	title       string
	description string //необязательно
	status      Status
	priority    Priority
	dueAt       *time.Time // дедлайн, необязательный 
}

// NewTask — создает новую задачу, с базовыми полями (id поле трогаем если только знаем, что ничего плохого не будет!)
func NewTask(title, description string) (*Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, errors.New("title is empty")
	}
	now := time.Now()
	return &Task{
		id:          generateID(),
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

// SetID — может пригодиться при пересоздании или нумерации
func (t *Task) SetID(id ID) {
	t.id = id
}

func (t *Task) TypeName() string { return "task" }

// Геттеры и всякое для получения
func (t *Task) ID() ID                  { return t.id }
func (t *Task) Title() string           { return t.title }
func (t *Task) Description() string     { return t.description }
func (t *Task) Status() Status          { return t.status }
func (t *Task) Priority() Priority      { return t.priority }
func (t *Task) DueAt() *time.Time       { return t.dueAt }
func (t *Task) CreatedAt() time.Time    { return t.createdAt }
func (t *Task) UpdatedAt() time.Time    { return t.updatedAt }
func (t *Task) CompletedAt() *time.Time { return t.completedAt }

// Меняет заголовок и трогает updatedAt
func (t *Task) SetTitle(title string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return errors.New("title is empty")
	}
	t.title = title
	t.touch()
	return nil
}

// Меняет описание, ничего особенного
func (t *Task) SetDescription(desc string) {
	t.description = strings.TrimSpace(desc)
	t.touch()
}

// Меняет статус, если done — ставит отметку о завершении
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

// Меняет приоритет задачи (1,2,3)
func (t *Task) SetPriority(p Priority) error {
	if !p.Valid() {
		return fmt.Errorf("invalid priority: %d", p)
	}
	t.priority = p
	t.touch()
	return nil
}

// Задает срок задачи (как бы дедлайн)
func (t *Task) SetDueAt(d time.Time) {
	dd := d
	t.dueAt = &dd
	t.touch()
}

// Убирает срок если решили без него
func (t *Task) ClearDue() {
	t.dueAt = nil
	t.touch()
}

// TaskDTO — используется чтобы сохранять задачу в JSON
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

// FromDTO — восстанавливает задачу из JSON
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
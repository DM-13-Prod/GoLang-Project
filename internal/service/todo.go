package service

import (
	"context"
	"sort"
	"strconv"
	"time"

	"todo/internal/model"
)

func logEvent(op string, id model.ID, before, after *model.TaskDTO) {
	if Logger == nil {
		return
	}
	_ = Logger.LogEvent(context.Background(), Event{
		Op:     op,
		TaskID: id,
		At:     time.Now(),
		Before: before,
		After:  after,
	})
}

type Service struct {
	store  Store
	tasks  map[model.ID]*model.Task
	nextID model.ID
}

func New(store Store) (*Service, error) {
	s := &Service{
		store: store,
		tasks: make(map[model.ID]*model.Task),
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Service) load() error {
	records, err := s.store.Load()
	if err != nil {
		return err
	}
	var maxID model.ID = 0
	for _, r := range records {
		t, err := model.FromDTO(r)
		if err != nil {
			continue
		}
		s.tasks[t.ID()] = t
		if t.ID() > maxID {
			maxID = t.ID()
		}
	}
	if maxID < 1 {
		s.nextID = 1
	} else {
		s.nextID = maxID + 1
	}
	return nil
}

func (s *Service) persist() error {
	all := make([]model.TaskDTO, 0, len(s.tasks))
	for _, t := range s.tasks {
		all = append(all, t.ToDTO())
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].CreatedAt.Before(all[j].CreatedAt)
	})
	return s.store.Save(all)
}

func (s *Service) Add(title, desc string, p model.Priority, due *time.Time) (model.ID, error) {
	t, err := model.NewTask(title, desc)
	if err != nil {
		return 0, err
	}
	_ = t.SetPriority(p)
	if due != nil {
		t.SetDueAt(*due)
	}
	t.SetID(s.nextID)
	s.tasks[t.ID()] = t
	s.nextID++
	if err := s.persist(); err != nil {
		return 0, err
	}
	after := t.ToDTO()
	logEvent("add", t.ID(), nil, &after)
	return t.ID(), nil
}

// RenumberIDs — перенумеровывает все задачи в порядке CreatedAt: 1..N
func (s *Service) RenumberIDs() error {
	list := s.List(nil) // уже отсортировано по CreatedAt
	newMap := make(map[model.ID]*model.Task, len(list))
	var id model.ID = 1
	for _, t := range list {
		t.SetID(id)
		newMap[id] = t
		id++
	}
	s.tasks = newMap
	s.nextID = id
	if err := s.persist(); err != nil {
		return err
	}
	logEvent("renumber_ids", 0, nil, nil)
	return nil
}

func (s *Service) List(filter *model.Status) []*model.Task {
	result := make([]*model.Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		if filter != nil && t.Status() != *filter {
			continue
		}
		result = append(result, t)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt().Before(result[j].CreatedAt())
	})
	return result
}

func (s *Service) UpdateTitle(id model.ID, title string) error {
	t, ok := s.tasks[id]
	if !ok {
		return errNotFound(id)
	}
	before := t.ToDTO()
	if err := t.SetTitle(title); err != nil {
		return err
	}
	if err := s.persist(); err != nil {
		return err
	}
	after := t.ToDTO()
	logEvent("update_title", id, &before, &after)
	return nil
}

func (s *Service) UpdateDesc(id model.ID, desc string) error {
	t, ok := s.tasks[id]
	if !ok {
		return errNotFound(id)
	}
	before := t.ToDTO()
	t.SetDescription(desc)
	if err := s.persist(); err != nil {
		return err
	}
	after := t.ToDTO()
	logEvent("update_desc", id, &before, &after)
	return nil
}

func (s *Service) SetStatus(id model.ID, st model.Status) error {
	t, ok := s.tasks[id]
	if !ok {
		return errNotFound(id)
	}
	before := t.ToDTO()
	if err := t.SetStatus(st); err != nil {
		return err
	}
	if err := s.persist(); err != nil {
		return err
	}
	after := t.ToDTO()
	logEvent("set_status", id, &before, &after)
	return nil
}

func (s *Service) SetPriority(id model.ID, p model.Priority) error {
	t, ok := s.tasks[id]
	if !ok {
		return errNotFound(id)
	}
	before := t.ToDTO()
	if err := t.SetPriority(p); err != nil {
		return err
	}
	if err := s.persist(); err != nil {
		return err
	}
	after := t.ToDTO()
	logEvent("set_priority", id, &before, &after)
	return nil
}

func (s *Service) SetDue(id model.ID, due time.Time) error {
	t, ok := s.tasks[id]
	if !ok {
		return errNotFound(id)
	}
	before := t.ToDTO()
	t.SetDueAt(due)
	if err := s.persist(); err != nil {
		return err
	}
	after := t.ToDTO()
	logEvent("set_due", id, &before, &after)
	return nil
}

func (s *Service) ClearDue(id model.ID) error {
	t, ok := s.tasks[id]
	if !ok {
		return errNotFound(id)
	}
	before := t.ToDTO()
	t.ClearDue()
	if err := s.persist(); err != nil {
		return err
	}
	after := t.ToDTO()
	logEvent("clear_due", id, &before, &after)
	return nil
}

func (s *Service) Delete(id model.ID) error {
	t, ok := s.tasks[id]
	if !ok {
		return errNotFound(id)
	}
	before := t.ToDTO()
	delete(s.tasks, id)
	if err := s.persist(); err != nil {
		return err
	}
	logEvent("delete", id, &before, nil)
	return nil
}

type notFound struct{ id model.ID }

func (e notFound) Error() string { return "task not found: " + strconv.FormatInt(int64(e.id), 10) }

func errNotFound(id model.ID) error { return notFound{id: id} }

// Гарантируем, что Service реализует TaskUseCase
var _ TaskUseCase = (*Service)(nil)
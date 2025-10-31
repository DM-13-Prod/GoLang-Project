package service

import (
	"sort"
	"strconv"
	"time"

	"todo/internal/model"
	"todo/internal/repository"
)

type Service struct {
	store  *repository.JSONStore
	tasks  map[model.ID]*model.Task
	nextID model.ID
}

func New(store *repository.JSONStore) (*Service, error) {
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
	return t.ID(), s.persist()
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
	return s.persist()
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
	if err := t.SetTitle(title); err != nil {
		return err
	}
	return s.persist()
}

func (s *Service) UpdateDesc(id model.ID, desc string) error {
	t, ok := s.tasks[id]
	if !ok {
		return errNotFound(id)
	}
	t.SetDescription(desc)
	return s.persist()
}

func (s *Service) SetStatus(id model.ID, st model.Status) error {
	t, ok := s.tasks[id]
	if !ok {
		return errNotFound(id)
	}
	if err := t.SetStatus(st); err != nil {
		return err
	}
	return s.persist()
}

func (s *Service) SetPriority(id model.ID, p model.Priority) error {
	t, ok := s.tasks[id]
	if !ok {
		return errNotFound(id)
	}
	if err := t.SetPriority(p); err != nil {
		return err
	}
	return s.persist()
}

func (s *Service) SetDue(id model.ID, due time.Time) error {
	t, ok := s.tasks[id]
	if !ok {
		return errNotFound(id)
	}
	t.SetDueAt(due)
	return s.persist()
}

func (s *Service) ClearDue(id model.ID) error {
	t, ok := s.tasks[id]
	if !ok {
		return errNotFound(id)
	}
	t.ClearDue()
	return s.persist()
}

func (s *Service) Delete(id model.ID) error {
	if _, ok := s.tasks[id]; !ok {
		return errNotFound(id)
	}
	delete(s.tasks, id)
	return s.persist()
}

type notFound struct{ id model.ID }

func (e notFound) Error() string { return "task not found: " + strconv.FormatInt(int64(e.id), 10) }

func errNotFound(id model.ID) error { return notFound{id: id} }

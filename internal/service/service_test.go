package service_test

import (
	"errors"
	"testing"
	"time"

	"todo/internal/model"
	"todo/internal/service"
)

// фейковое хранилище для тестов
type fakeStore struct {
	items    []model.TaskDTO
	loadErr  error
	saveErr  error
	saveCall int
}

func (f *fakeStore) Load() ([]model.TaskDTO, error) {
	if f.loadErr != nil {
		return nil, f.loadErr
	}
	cp := make([]model.TaskDTO, len(f.items))
	copy(cp, f.items)
	return cp, nil
}

func (f *fakeStore) Save(items []model.TaskDTO) error {
	f.saveCall++
	if f.saveErr != nil {
		return f.saveErr
	}
	cp := make([]model.TaskDTO, len(items))
	copy(cp, items)
	f.items = cp
	return nil
}

func mustNewService(t *testing.T, initial []model.TaskDTO) (*service.Service, *fakeStore) {
	t.Helper()
	fs := &fakeStore{items: initial}
	svc, err := service.New(fs)
	if err != nil {
		t.Fatalf("service.New error: %v", err)
	}
	return svc, fs
}

func findTaskByID(list []*model.Task, id model.ID) *model.Task {
	for _, t := range list {
		if t.ID() == id {
			return t
		}
	}
	return nil
}

func TestNew_LoadError(t *testing.T) {
	fs := &fakeStore{loadErr: errors.New("boom")}
	_, err := service.New(fs)
	if err == nil {
		t.Fatal("expected error on load, got nil")
	}
}

func TestAdd_And_List(t *testing.T) {
	svc, fs := mustNewService(t, nil)

	id, err := svc.Add("A", "desc", model.PriorityHigh, nil)
	if err != nil {
		t.Fatalf("Add error: %v", err)
	}
	if fs.saveCall != 1 {
		t.Fatalf("expected 1 save, got %d", fs.saveCall)
	}
	got := svc.List(nil)
	if len(got) != 1 {
		t.Fatalf("expected 1 task, got %d", len(got))
	}
	if got[0].ID() != id || got[0].Title() != "A" || got[0].Priority() != model.PriorityHigh {
		t.Fatalf("unexpected task: %+v", got[0])
	}
}

func TestAdd_EmptyTitle(t *testing.T) {
	svc, fs := mustNewService(t, nil)
	_, err := svc.Add("", "", model.PriorityMedium, nil)
	if err == nil {
		t.Fatal("expected error for empty title, got nil")
	}
	if fs.saveCall != 0 {
		t.Fatalf("expected 0 saves, got %d", fs.saveCall)
	}
}

func TestUpdateTitle_Desc(t *testing.T) {
	svc, _ := mustNewService(t, nil)
	id, _ := svc.Add("A", "x", model.PriorityMedium, nil)

	if err := svc.UpdateTitle(id, "B"); err != nil {
		t.Fatalf("UpdateTitle err: %v", err)
	}
	if err := svc.UpdateDesc(id, "Y"); err != nil {
		t.Fatalf("UpdateDesc err: %v", err)
	}
	got := findTaskByID(svc.List(nil), id)
	if got == nil || got.Title() != "B" || got.Description() != "Y" {
		t.Fatalf("unexpected task: %+v", got)
	}
}

func TestUpdateTitle_Empty(t *testing.T) {
	svc, _ := mustNewService(t, nil)
	id, _ := svc.Add("A", "", model.PriorityMedium, nil)
	if err := svc.UpdateTitle(id, ""); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSetStatus_Transitions(t *testing.T) {
	svc, _ := mustNewService(t, nil)
	id, _ := svc.Add("A", "", model.PriorityLow, nil)

	// invalid
	if err := svc.SetStatus(id, model.Status("bad")); err == nil {
		t.Fatal("expected error for invalid status")
	}

	// done -> CompletedAt выставлен
	if err := svc.SetStatus(id, model.StatusDone); err != nil {
		t.Fatalf("SetStatus done err: %v", err)
	}
	tk := findTaskByID(svc.List(nil), id)
	if tk.CompletedAt() == nil {
		t.Fatal("expected CompletedAt to be set")
	}

	// выход из done -> CompletedAt очищен
	if err := svc.SetStatus(id, model.StatusInProgress); err != nil {
		t.Fatalf("SetStatus in_progress err: %v", err)
	}
	tk = findTaskByID(svc.List(nil), id)
	if tk.CompletedAt() != nil {
		t.Fatal("expected CompletedAt to be nil after leaving done")
	}
}

func TestSetPriority(t *testing.T) {
	svc, _ := mustNewService(t, nil)
	id, _ := svc.Add("A", "", model.PriorityLow, nil)

	if err := svc.SetPriority(id, model.PriorityHigh); err != nil {
		t.Fatalf("SetPriority err: %v", err)
	}
	if got := findTaskByID(svc.List(nil), id).Priority(); got != model.PriorityHigh {
		t.Fatalf("expected high, got %v", got)
	}

	if err := svc.SetPriority(id, 0); err == nil {
		t.Fatal("expected error for invalid priority")
	}
}

func TestDue_ClearDue(t *testing.T) {
	svc, _ := mustNewService(t, nil)
	id, _ := svc.Add("A", "", model.PriorityMedium, nil)

	d := time.Now().Add(24 * time.Hour).Truncate(24 * time.Hour)
	if err := svc.SetDue(id, d); err != nil {
		t.Fatalf("SetDue err: %v", err)
	}
	tk := findTaskByID(svc.List(nil), id)
	if tk.DueAt() == nil || !tk.DueAt().Equal(d) {
		t.Fatalf("unexpected due: %v", tk.DueAt())
	}
	if err := svc.ClearDue(id); err != nil {
		t.Fatalf("ClearDue err: %v", err)
	}
	if findTaskByID(svc.List(nil), id).DueAt() != nil {
		t.Fatal("expected due to be nil after ClearDue")
	}
}

func TestDelete_NotFound(t *testing.T) {
	svc, _ := mustNewService(t, nil)
	if err := svc.Delete(42); err == nil {
		t.Fatal("expected not found")
	}
}

func TestList_Filter(t *testing.T) {
	svc, _ := mustNewService(t, nil)
	id1, _ := svc.Add("A", "", model.PriorityLow, nil)
	id2, _ := svc.Add("B", "", model.PriorityLow, nil)
	_ = svc.SetStatus(id2, model.StatusDone)

	all := svc.List(nil)
	if len(all) != 2 {
		t.Fatalf("expected 2, got %d", len(all))
	}
	f := model.StatusDone
	onlyDone := svc.List(&f)
	if len(onlyDone) != 1 || onlyDone[0].ID() != id2 {
		t.Fatalf("expected only done id=%d, got %+v", id2, onlyDone)
	}
	_ = id1
}

func TestRenumberIDs(t *testing.T) {
	now := time.Now()
	initial := []model.TaskDTO{
		{
			ID:        10,
			Title:     "old",
			Status:    model.StatusNew,
			Priority:  model.PriorityMedium,
			CreatedAt: now.Add(-2 * time.Hour),
			UpdatedAt: now.Add(-2 * time.Hour),
		},
		{
			ID:        7,
			Title:     "newer",
			Status:    model.StatusNew,
			Priority:  model.PriorityMedium,
			CreatedAt: now.Add(-1 * time.Hour),
			UpdatedAt: now.Add(-1 * time.Hour),
		},
	}
	svc, fs := mustNewService(t, initial)

	if err := svc.RenumberIDs(); err != nil {
		t.Fatalf("RenumberIDs err: %v", err)
	}
	if fs.saveCall == 0 {
		t.Fatal("expected persist on renumber")
	}
	list := svc.List(nil)
	if len(list) != 2 {
		t.Fatalf("expected 2, got %d", len(list))
	}
	if list[0].ID() != 1 || list[0].Title() != "old" {
		t.Fatalf("first should be id=1, title=old, got id=%d, title=%s", list[0].ID(), list[0].Title())
	}
	if list[1].ID() != 2 || list[1].Title() != "newer" {
		t.Fatalf("second should be id=2, title=newer, got id=%d, title=%s", list[1].ID(), list[1].Title())
	}
}

func TestPersistErrorsBubbleUp(t *testing.T) {
	fs := &fakeStore{}
	svc, _ := service.New(fs)

	fs.saveErr = errors.New("save failed")
	if _, err := svc.Add("A", "", model.PriorityLow, nil); err == nil {
		t.Fatal("expected error from Save")
	}
}
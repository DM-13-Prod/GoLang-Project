package repository

import (
	"todo/internal/model"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"context"
)

// Тут просто хранилища задач по приоритету.
// Разделение нужно чтоб потом их быстро доставать.
var (
	LowPriorityTasks    []*model.Task
	MediumPriorityTasks []*model.Task
	HighPriorityTasks   []*model.Task

	muLow, muMed, muHigh sync.Mutex //Мьютексы

	// Пути к файлам, где будем сохранять распределённые задачи
	lowFile = filepath.Join("cmd", "data", "low_tasks.json")
	mediumFile = filepath.Join("cmd", "data", "medium_tasks.json")
	highFile = filepath.Join("cmd", "data", "high_tasks.json")
)

// init — при старте вытаскивает данные с диска, если они у нас уже были.
func init() {
	LowPriorityTasks = loadPriorityTasks(lowFile)
	MediumPriorityTasks = loadPriorityTasks(mediumFile)
	HighPriorityTasks = loadPriorityTasks(highFile)
}

func (s *PostgresStore) UpdateTaskAndLog(ctx context.Context, id model.ID, status model.Status) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		UPDATE tasks SET status=$1, updated_at=now() WHERE id=$2
	`, status, id)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO audit_logs (task_id, operation) VALUES ($1, $2)
	`, id, "update_status")
	if err != nil {
		return err
	}

	return tx.Commit()
}

// Distribute — раскидывает задачу по нужному списку, в зависимости от приоритета
func Distribute(e Entity) {
	switch v := e.(type) {
	case *model.Task:
		switch v.Priority() {
		case model.PriorityLow:
			muLow.Lock()
			LowPriorityTasks = append(LowPriorityTasks, v)
			savePriorityTasks(lowFile, LowPriorityTasks)
			muLow.Unlock()

		case model.PriorityMedium:
			muMed.Lock()
			MediumPriorityTasks = append(MediumPriorityTasks, v)
			savePriorityTasks(mediumFile, MediumPriorityTasks)
			muMed.Unlock()

		case model.PriorityHigh:
			muHigh.Lock()
			HighPriorityTasks = append(HighPriorityTasks, v)
			savePriorityTasks(highFile, HighPriorityTasks)
			muHigh.Unlock()

		default:
			fmt.Println("неизвестный приоритет:", v.Priority())
		}

	default:
		fmt.Println("репозиторий: неизвестный тип:", v)
	}
}

// loadPriorityTasks — читает json‑файл и восстанавливает []*model.Task
func loadPriorityTasks(path string) []*model.Task {
	_ = os.MkdirAll(filepath.Dir(path), 0o755)

	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return []*model.Task{}
	}

	var raw []model.TaskDTO
	if err := json.Unmarshal(data, &raw); err != nil {
		fmt.Println("ошибка чтения", path, ":", err)
		return []*model.Task{}
	}

	var result []*model.Task
	for _, dto := range raw {
		t, err := model.FromDTO(dto)
		if err == nil {
			result = append(result, t)
		}
	}
	return result
}

// savePriorityTasks — сериализует и сохраняет список задач в файл
func savePriorityTasks(path string, tasks []*model.Task) {
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	items := make([]model.TaskDTO, 0, len(tasks))
	for _, t := range tasks {
		items = append(items, t.ToDTO())
	}
	raw, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		fmt.Println("ошибка сериализации", path, ":", err)
		return
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		fmt.Println("ошибка записи", path, ":", err)
		return
	}
	if err := os.Rename(tmp, path); err != nil {
		fmt.Println("ошибка переименования", path, ":", err)
	}
}
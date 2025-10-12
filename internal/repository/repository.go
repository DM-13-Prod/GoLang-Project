package repository

import (
	"cmd/internal/model"
	"fmt"
	"sync"
)

// Тут просто хранилища задач по приоритету.
// Разделение нужно чтоб потом их быстро доставать.
var (
	LowPriorityTasks    []*model.Task
	MediumPriorityTasks []*model.Task
	HighPriorityTasks   []*model.Task
	muLow, muMed, muHigh sync.Mutex //Мьютексы
)

// Distribute — раскидывает задачу по нужному списку, в зависимости от приоритета
func Distribute(e Entity) {
	switch v := e.(type) {
	case *model.Task:
		switch v.Priority() {
		case model.PriorityLow:
			muLow.Lock()
			LowPriorityTasks = append(LowPriorityTasks, v)
			muLow.Unlock()
		case model.PriorityMedium:
			muMed.Lock()
			MediumPriorityTasks = append(MediumPriorityTasks, v)
			muMed.Unlock()
		case model.PriorityHigh:
			muHigh.Lock()
			HighPriorityTasks = append(HighPriorityTasks, v)
			muHigh.Unlock()
		default:
			fmt.Println("неизвестный приоритет:", v.Priority())
		}
	default:
		fmt.Println("репозиторий: неизвестный тип:", v)
	}
}


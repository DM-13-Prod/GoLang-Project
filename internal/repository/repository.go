package repository

import (
	"cmd/internal/model"
	"fmt"
)

// Тут просто хранилища задач по приоритету.
// Разделение нужно чтоб потом их быстро доставать.
var (
	LowPriorityTasks    []*model.Task
	MediumPriorityTasks []*model.Task
	HighPriorityTasks   []*model.Task
)

// Distribute — раскидывает задачу по нужному списку, в зависимости от приоритета
func Distribute(e Entity) {
	switch v := e.(type) {
	case *model.Task:
		switch v.Priority() {
		case model.PriorityLow:
			LowPriorityTasks = append(LowPriorityTasks, v)
		case model.PriorityMedium:
			MediumPriorityTasks = append(MediumPriorityTasks, v)
		case model.PriorityHigh:
			HighPriorityTasks = append(HighPriorityTasks, v)
		default:
			fmt.Println("неизвестный приоритет:", v.Priority())
		}
	default:
		// если что-то вообще непонятное попадет — просто выведем
		fmt.Println("репозиторий: неизвестный тип сущности:", v)
	}
}
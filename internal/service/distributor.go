package service

import (
	"cmd/internal/model"
	"context"
	"cmd/internal/repository"
	"fmt"
	"math/rand"
	"time"
)

var PrintLogs = true // можно включать/выключать подробный вывод

// DistributeNewTasksPeriodically — создаёт задачи (использует бизнес-логику) каждые interval секунд, реализация ДЗ из 12 блока.
// Делает это пока не придёт сигнал стопа.
func DistributeNewTasksPeriodically(interval time.Duration, ctx context.Context) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// создаём задачу-автомат
			t, _ := model.NewTask(
				fmt.Sprintf("Auto-Task-%d", time.Now().Unix()),
				"автоматически созданная задача для тестов",
			)

			// назначаем случайный приоритет 1..3
			t.SetPriority(model.Priority(rand.Intn(3) + 1))

			// передаём в слой repository
			repository.Distribute(t)

			if DebugMode {
				// просто пишет что произошло, типа лог
				fmt.Printf("[дистрибьютор] создана задача %v с приоритетом %v\n",
					t.Title(), t.Priority())
			}

		case <-ctx.Done():
			fmt.Println("[дистрибьютор] остановлен")
			return
		}
	}
}

func GenerateTasks(out chan<- *model.Task, interval time.Duration, ctx context.Context) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			t, _ := model.NewTask(
				fmt.Sprintf("ChTask-%d", time.Now().UnixNano()),
				"создана через канал",
			)
			t.SetPriority(model.Priority(rand.Intn(3) + 1))
			if DebugMode {
				fmt.Printf("[генератор] → %s (%v)\n", t.Title(), t.Priority())
			}
			out <- t
		case <-ctx.Done():
			fmt.Println("[гeнератор] остановлен")
			close(out)
			return
		}
	}
}
func DistributeFromChannel(in <-chan *model.Task, ctx context.Context) {
	for {
		select {
		case t, ok := <-in:
			if !ok {
				fmt.Println("[канальный дистрибьютор] остановлен (канал закрыт)")
				return
			}
			repository.Distribute(t)
			if DebugMode {
				fmt.Printf("[канальный дистрибьютор] принял %s → %v\n",
					t.Title(), t.Priority())
			}
		case <-ctx.Done():
			fmt.Println("[канальный дистрибьютор] остановлен")
			return
		}
	}
}

// LogTaskAdditions — каждые 200 мс проверяет, не добавили ли чего нового
func LogTaskAdditions(interval time.Duration, ctx context.Context) {
    prevLow, prevMed, prevHigh := 0, 0, 0
    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            curLow := len(repository.LowPriorityTasks)
            curMed := len(repository.MediumPriorityTasks)
            curHigh := len(repository.HighPriorityTasks)

            if !DebugMode {
                //просто обновляем счётчики без вывода
                prevLow, prevMed, prevHigh = curLow, curMed, curHigh
                continue
            }

            if curLow > prevLow {
                diff := curLow - prevLow
                fmt.Printf("[логгер] добавлено %d низкоприоритетных задач\n", diff)
                prevLow = curLow
            }
            if curMed > prevMed {
                diff := curMed - prevMed
                fmt.Printf("[логгер] добавлено %d средних задач\n", diff)
                prevMed = curMed
            }
            if curHigh > prevHigh {
                diff := curHigh - prevHigh
                fmt.Printf("[логгер] добавлено %d высокоприоритетных задач\n", diff)
                prevHigh = curHigh
            }

        case <-ctx.Done():
            if DebugMode {
                fmt.Println("[логгер] остановлен")
            }
            return
        }
    }
}
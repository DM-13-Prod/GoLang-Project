package service

import (
	"cmd/internal/model"
	"cmd/internal/repository"
	"fmt"
	"math/rand"
	"time"
)

var PrintLogs = true // можно включать/выключать подробный вывод

// DistributeNewTasksPeriodically — создаёт задачи (использует бизнес-логику) каждые interval секунд, реализация ДЗ из 12 блока.
// Делает это пока не придёт сигнал стопа.
func DistributeNewTasksPeriodically(interval time.Duration, stop <-chan struct{}) {
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

		case <-stop:
			fmt.Println("[дистрибьютор] остановлен")
			return
		}
	}
}
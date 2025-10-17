package main

import (
	"bufio"
	"fmt"
	"context"
	"os/signal"
	"sync"
	"syscall"
	"os"
	"strconv"
	"strings"
	"time"

	"cmd/internal/model"
	"cmd/internal/service"
	"cmd/internal/storage"
	"cmd/internal/repository"
)

func main() {
	// Путь к JSON-файлу где всё хранится.
	storePath := os.Getenv("TASKS_FILE")
	if storePath == "" {
		storePath = "data/tasks.json"
	}


	svc, err := service.New(storage.NewJSONStore(storePath))
	if err != nil {
		fmt.Println("init error:", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-sigCh
			fmt.Println("\nПолучен сигнал завершения, останавливаем сервис...")
			cancel()
		}()

		var wg sync.WaitGroup
		wg.Add(4)

		// Старый авто-дистрибьютор
		go func() {
			defer wg.Done()
			service.DistributeNewTasksPeriodically(10*time.Second, ctx)
		}()

		// Новый конкурентный набор
		ch := make(chan *model.Task, 10)

		go func() {
			defer wg.Done()
			service.GenerateTasks(ch, 5*time.Second, ctx)
		}()
		go func() {
			defer wg.Done()
			service.DistributeFromChannel(ch, ctx)
		}()
		go func() {
			defer wg.Done()
			service.LogTaskAdditions(200*time.Millisecond, ctx)
		}()

	in := bufio.NewScanner(os.Stdin)

	for {
		fmt.Println()
		fmt.Println("== TODO / Задачник ==")

		fmt.Println("1)  Добавить задачу")
		fmt.Println("2)  Список всех задач")
		fmt.Println("3)  Список по статусу")
		fmt.Println("11) Показать задачу")
		fmt.Println()
		fmt.Println("4)  Обновить заголовок/описание")
		fmt.Println("5)  Поменять статус")
		fmt.Println("6)  Поменять приоритет")
		fmt.Println("7)  Установить/очистить срок (Due)")
		fmt.Println("8)  Удалить задачу")
		fmt.Println()
		fmt.Println(" Расширенные служебные функции:")
		fmt.Println("10) Перенумеровать ID (1..N)")
		fmt.Println("12) Показать распределённые задачи (repository)")
		fmt.Println("13) Переключить Debug‑режим")
		fmt.Println()
		fmt.Println("9)  Выход")
		fmt.Print("Выбор: ")

		choice := readLine(in)
		switch choice {

		case "1":
			handleAdd(in, svc)
		case "2":
			printTasks(svc.List(nil))
		case "3":
			st, ok := askStatus(in)
			if !ok {
				fmt.Println("отмена")
				continue
			}
			printTasks(svc.List(&st))
		case "4":
			handleUpdateText(in, svc)
		case "5":
			handleStatus(in, svc)
		case "6":
			handlePriority(in, svc)
		case "7":
			handleDue(in, svc)
		case "8":
			handleDelete(in, svc)
		case "9":
			cancel()
			wg.Wait()
			fmt.Println("Все горутины завершены. Пока!")
			return
		case "10":
			fmt.Print("Перенумеровать все ID в 1..N? (y/N): ")
			ans := strings.ToLower(strings.TrimSpace(readLine(in)))
			if ans != "y" && ans != "yes" {
				fmt.Println("отмена")
				break
			}
			if err := svc.RenumberIDs(); err != nil {
				fmt.Println("ошибка:", err)
			} else {
				fmt.Println("OK: ID перенумерованы")
				printTasks(svc.List(nil))
			}
		case "11":
			id, ok := askID(in)
			if !ok {
				break
			}
			var found *model.Task
			for _, t := range svc.List(nil) {
				if t.ID() == id {
					found = t
					break
				}
			}
			if found == nil {
				fmt.Println("не найдено")
				break
			}
			printTaskDetails(found)
		case "12":
			fmt.Println("= низкий приоритет =")
			printTasks(repository.LowPriorityTasks)
			fmt.Println("= средний приоритет =")
			printTasks(repository.MediumPriorityTasks)
			fmt.Println("= высокий приоритет =")
			printTasks(repository.HighPriorityTasks)
		case "13":
			service.DebugMode = !service.DebugMode
			if service.DebugMode {
				fmt.Println("Debug‑режим включён (подробный вывод)")
			} else {
				fmt.Println("Debug‑режим выключен")
			}
		default:
			fmt.Println("неизвестная команда")
		}
	}
}

// Выводит детали одной задачи
func printTaskDetails(t *model.Task) {
	due := "-"
	if d := t.DueAt(); d != nil {
		due = d.Format("02-01-2006")
	}
	fmt.Printf("Title: %s\n", t.Title())
	fmt.Printf("Status: %s\n", t.Status())
	fmt.Printf("Priority: %s\n", prioText(t.Priority()))
	fmt.Printf("Created: %s\n", t.CreatedAt().Format("2006-01-02 15:04"))
	fmt.Printf("Updated: %s\n", t.UpdatedAt().Format("2006-01-02 15:04"))
	fmt.Printf("Due: %s\n", due)
	fmt.Println("Description:")
	fmt.Println(t.Description())
}

// Создает новую задачу через консоль
func handleAdd(in *bufio.Scanner, svc *service.Service) {
	fmt.Print("Заголовок: ")
	title := strings.TrimSpace(readLine(in))
	if title == "" {
		fmt.Println("пустой заголовок")
		return
	}
	fmt.Print("Описание (необязательно): ")
	desc := strings.TrimSpace(readLine(in))
	p := askPriority(in)

	var due *time.Time
	fmt.Print("Дедлайн (DD-MM-YYYY): ")
	if s := strings.TrimSpace(readLine(in)); s != "" {
		d, err := parseDMYDate(s)
		if err != nil {
			fmt.Println("дата некорректна:", err)
		} else {
			due = &d
		}
	}

	id, err := svc.Add(title, desc, p, due)
	if err != nil {
		fmt.Println("ошибка добавления:", err)
		return
	}
	fmt.Println("OK, id =", id)
}

func handleUpdateText(in *bufio.Scanner, svc *service.Service) {
	id, ok := askID(in)
	if !ok {
		return
	}
	fmt.Print("Новый заголовок (пусто - пропустить): ")
	title := strings.TrimSpace(readLine(in))
	fmt.Print("Новое описание (пусто - пропустить): ")
	desc := strings.TrimSpace(readLine(in))

	if title != "" {
		if err := svc.UpdateTitle(id, title); err != nil {
			fmt.Println("ошибка:", err)
			return
		}
	}
	if desc != "" {
		if err := svc.UpdateDesc(id, desc); err != nil {
			fmt.Println("ошибка:", err)
			return
		}
	}
	fmt.Println("OK")
}

func handleStatus(in *bufio.Scanner, svc *service.Service) {
	id, ok := askID(in)
	if !ok {
		return
	}
	st, ok := askStatus(in)
	if !ok {
		return
	}
	if err := svc.SetStatus(id, st); err != nil {
		fmt.Println("ошибка:", err)
		return
	}
	fmt.Println("OK")
}

func handlePriority(in *bufio.Scanner, svc *service.Service) {
	id, ok := askID(in)
	if !ok {
		return
	}
	p := askPriority(in)
	if err := svc.SetPriority(id, p); err != nil {
		fmt.Println("ошибка:", err)
		return
	}
	fmt.Println("OK")
}

func handleDue(in *bufio.Scanner, svc *service.Service) {
	id, ok := askID(in)
	if !ok {
		return
	}
	fmt.Print("Дата (DD-MM-YYYY) или пусто для очистки: ")
	raw := strings.TrimSpace(readLine(in))
	if raw == "" {
		if err := svc.ClearDue(id); err != nil {
			fmt.Println("ошибка:", err)
		} else {
			fmt.Println("OK (очищено)")
		}
		return
	}
	d, err := parseDMYDate(raw)
	if err != nil {
		fmt.Println("дата некорректна:", err)
		return
	}
	if err := svc.SetDue(id, d); err != nil {
		fmt.Println("ошибка:", err)
		return
	}
	fmt.Println("OK")
}

func handleDelete(in *bufio.Scanner, svc *service.Service) {
	id, ok := askID(in)
	if !ok {
		return
	}
	if err := svc.Delete(id); err != nil {
		fmt.Println("ошибка:", err)
		return
	}
	fmt.Println("OK (удалено)")
}

func askID(in *bufio.Scanner) (model.ID, bool) {
	fmt.Print("ID: ")
	raw := strings.TrimSpace(readLine(in))
	if raw == "" {
		fmt.Println("отмена")
		return 0, false
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		fmt.Println("не число")
		return 0, false
	}
	return model.ID(v), true
}

func askStatus(in *bufio.Scanner) (model.Status, bool) {
	fmt.Println("Статус:")
	fmt.Println("  1) new")
	fmt.Println("  2) in_progress")
	fmt.Println("  3) done")
	fmt.Println("  4) paused")
	fmt.Println("  5) canceled")
	fmt.Print("Выбор: ")
	choice := readLine(in)
	switch choice {
	case "1":
		return model.StatusNew, true
	case "2":
		return model.StatusInProgress, true
	case "3":
		return model.StatusDone, true
	case "4":
		return model.StatusPaused, true
	case "5":
		return model.StatusCanceled, true
	default:
		fmt.Println("отмена")
		return "", false
	}
}

func askPriority(in *bufio.Scanner) model.Priority {
	fmt.Println("Приоритет:")
	fmt.Println("  1) low")
	fmt.Println("  2) medium")
	fmt.Println("  3) high")
	fmt.Print("Выбор [2]: ")
	choice := strings.TrimSpace(readLine(in))
	switch choice {
	case "1":
		return model.PriorityLow
	case "3":
		return model.PriorityHigh
	default:
		return model.PriorityMedium
	}
}

// вывод всех задач таблицей
func printTasks(list []*model.Task) {
	if len(list) == 0 {
		fmt.Println("(пусто)")
		return
	}
	fmt.Println("ID | Title | Status | Prio | Created | Due | Desc")
	for _, t := range list {
		due := "-"
		if d := t.DueAt(); d != nil {
			due = d.Format("02-01-2006")
		}
		desc := trunc(t.Description(), 43)
		fmt.Printf("%d | %s | %s | %s | %s | %s | %s\n",
			t.ID(), t.Title(), t.Status(), prioText(t.Priority()),
			t.CreatedAt().Format("2006-01-02 15:04"),
			due, desc,
		)
	}
}

func trunc(s string, n int) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "-"
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

func prioText(p model.Priority) string {
	switch p {
	case model.PriorityLow:
		return "low"
	case model.PriorityHigh:
		return "high"
	default:
		return "medium"
	}
}

// парсит дату формата DD-MM-YYYY, типо чтобы нормально работало с человеком
func parseDMYDate(input string) (time.Time, error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return time.Time{}, fmt.Errorf("пустая дата")
	}

	digits := make([]rune, 0, len(s))
	for _, ch := range s {
		if ch >= '0' && ch <= '9' {
			digits = append(digits, ch)
		}
	}
	if len(digits) == 8 {
		day, _ := strconv.Atoi(string(digits[0:2]))
		month, _ := strconv.Atoi(string(digits[2:4]))
		year, _ := strconv.Atoi(string(digits[4:8]))
		return makeDateChecked(year, month, day)
	}

	s = strings.ReplaceAll(s, ".", "-")
	parts := strings.Split(s, "-")
	if len(parts) != 3 {
		return time.Time{}, fmt.Errorf("нужно DD-MM-YYYY или DDMMYYYY")
	}
	day, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
	month, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
	year, _ := strconv.Atoi(strings.TrimSpace(parts[2]))
	return makeDateChecked(year, month, day)
}

// проверяет дату и возвращает начало дня
func makeDateChecked(year, month, day int) (time.Time, error) {
	t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
	if t.Year() != year || int(t.Month()) != month || t.Day() != day {
		return time.Time{}, fmt.Errorf("некорректная дата")
	}
	return t, nil
}

func readLine(in *bufio.Scanner) string {
	if in.Scan() {
		return in.Text()
	}
	return ""
}
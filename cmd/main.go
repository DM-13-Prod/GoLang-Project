package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"cmd/internal/model"
	"cmd/internal/service"
	"cmd/internal/storage"
)

func main() {
	storePath := os.Getenv("TASKS_FILE")
	if storePath == "" {
		storePath = "data/tasks.json"
	}

	svc, err := service.New(storage.NewJSONStore(storePath))
	if err != nil {
		fmt.Println("init error:", err)
		os.Exit(1)
	}

	in := bufio.NewScanner(os.Stdin)
	for {
		fmt.Println()
		fmt.Println("== TODO / copybook ==")
		fmt.Println("1) Добавить задачу")
		fmt.Println("2) Список всех задач")
		fmt.Println("3) Список по статусу")
		fmt.Println("4) Обновить заголовок/описание")
		fmt.Println("5) Поменять статус")
		fmt.Println("6) Поменять приоритет")
		fmt.Println("7) Установить/очистить срок (due)")
		fmt.Println("8) Удалить задачу")
		fmt.Println("9) Выход")
		fmt.Println("10) Перенумеровать ID (1..N)")
		fmt.Println("11) Показать задачу")
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
			fmt.Println("Пока!")
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
			// без изменения сервиса — просто ищем в списке:
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

		default:
			fmt.Println("неизвестная команда")
		}
	}
}

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

func handleAdd(in *bufio.Scanner, svc *service.Service) {
	fmt.Print("Заголовок: ")
	title := strings.TrimSpace(readLine(in))
	if title == "" {
		fmt.Println("пустой заголовок")
		return
	}
	fmt.Print("Описание (опционально): ")
	desc := strings.TrimSpace(readLine(in))

	p := askPriority(in)

	var due *time.Time
	fmt.Print("Дедлайн (DD-MM-YYYY, можно DDMMYYYY или DD.MM.YYYY; пусто — без срока): ")
	if s := strings.TrimSpace(readLine(in)); s != "" {
		d, err := parseDMYDate(s)
		if err != nil {
			fmt.Println("дата некорректна, пропущено:", err)
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
	fmt.Print("Новый заголовок (пусто - оставить): ")
	title := strings.TrimSpace(readLine(in))
	fmt.Print("Новое описание (пусто - оставить): ")
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
	fmt.Print("Установить дату (DD-MM-YYYY, можно DDMMYYYY или DD.MM.YYYY) или пусто чтобы очистить: ")
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

func printTasks(list []*model.Task) {
	if len(list) == 0 {
		fmt.Println("(пусто)")
		return
	}
	fmt.Println("ID | Title | Status | Prio | Created | Due | Desc")
	for _, t := range list {
		due := "-"
		if d := t.DueAt(); d != nil {
			due = d.Format("02-01-2006") // DD-MM-YYYY
		}
		desc := trunc(t.Description(), 40) // укоротим до 40 символов
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

// parseDMYDate принимает DD-MM-YYYY, DDMMYYYY, DD.MM.YYYY и возвращает дату (локальную) на 00:00
func parseDMYDate(input string) (time.Time, error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return time.Time{}, fmt.Errorf("пустая дата")
	}

	// Вариант без разделителей: DDMMYYYY
	digits := make([]rune, 0, len(s))
	for _, ch := range s {
		if ch >= '0' && ch <= '9' {
			digits = append(digits, ch)
		}
	}
	if len(digits) == 8 { // DDMMYYYY
		day, err1 := strconv.Atoi(string(digits[0:2]))
		month, err2 := strconv.Atoi(string(digits[2:4]))
		year, err3 := strconv.Atoi(string(digits[4:8]))
		if err1 != nil || err2 != nil || err3 != nil {
			return time.Time{}, fmt.Errorf("не удалось разобрать дату")
		}
		return makeDateChecked(year, month, day)
	}

	// Варианты с разделителями: DD-MM-YYYY или DD.MM.YYYY
	s = strings.ReplaceAll(s, ".", "-")
	parts := strings.Split(s, "-")
	if len(parts) != 3 {
		return time.Time{}, fmt.Errorf("используйте DD-MM-YYYY, DDMMYYYY или DD.MM.YYYY")
	}
	day, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	month, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	year, err3 := strconv.Atoi(strings.TrimSpace(parts[2]))
	if err1 != nil || err2 != nil || err3 != nil {
		return time.Time{}, fmt.Errorf("не удалось разобрать дату")
	}
	if len(parts[2]) != 4 {
		return time.Time{}, fmt.Errorf("год должен быть в формате YYYY")
	}
	return makeDateChecked(year, month, day)
}

// makeDateChecked создаёт дату 00:00 в локальной зоне и валидирует корректность (например, отсекает 31-02-2025)
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

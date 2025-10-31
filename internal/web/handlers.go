// @title TODO API
// @version 1.0
// @description Simple task manager API example with JWT authorization
// @BasePath /api
//
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

package web

import (
	"todo/internal/model"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// LoginRequest — тело запроса для авторизации
type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// TaskCreateRequest — тело запроса при создании задачи
type TaskCreateRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    int    `json:"priority"`
	DueAt       string `json:"due_at"`
}

// TaskUpdateRequest — тело запроса при обновлении задачи
type TaskUpdateRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Priority    int    `json:"priority"`
	DueAt       string `json:"due_at"`
}

// Авторизация пользователя (возвращает JWT‑токен)
// handleLogin godoc
// @Summary      User login
// @Description  Authenticates user and returns JWT
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        credentials body LoginRequest true "User credentials"
// @Success      200 {object} map[string]string "token"
// @Failure      400 {string} string "invalid json"
// @Failure      401 {string} string "unauthorized"
// @Router       /login [post]
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var creds LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	login := os.Getenv("LOGIN")
	pass := os.Getenv("PASSWORD")
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "default_secret"
	}

	if creds.Login != login || creds.Password != pass {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"login": creds.Login,
		"exp":   time.Now().Add(2 * time.Hour).Unix(),
	})
	tokenStr, err := token.SignedString([]byte(secret))
	if err != nil {
		http.Error(w, "token error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": tokenStr})
}

// Middleware‑проверка JWT перед изменением данных
func (s *Server) withJWTAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}
		tokenStr := strings.TrimPrefix(auth, "Bearer ")
		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			secret = "default_secret"
		}

		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// Создание новой задачи
// handleCreateItem godoc
// @Summary      Create task
// @Description  Creates new task with optional due date
// @Tags         tasks
// @Accept       json
// @Produce      json
// @Param        data body TaskCreateRequest true "Task data"
// @Success      200 {object} map[string]interface{} "Created task ID"
// @Failure      400 {string} string "invalid json"
// @Failure      500 {string} string "server error"
// @Security     BearerAuth
// @Router       /item [post]
func (s *Server) handleCreateItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var dto TaskCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(dto.Title) == "" {
		http.Error(w, "title required", http.StatusBadRequest)
		return
	}

	var due *time.Time
	if dto.DueAt != "" {
		if t, err := time.Parse("2006-01-02", dto.DueAt); err == nil {
			due = &t
		}
	}

	id, err := s.svc.Add(dto.Title, dto.Description, model.Priority(dto.Priority), due)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"id": id})
}

// Возвращает список всех задач
// handleListItems godoc
// @Summary      List tasks
// @Description  Returns all tasks
// @Tags         tasks
// @Produce      json
// @Success      200 {array} model.TaskDTO
// @Router       /items [get]
func (s *Server) handleListItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	list := s.svc.List(nil)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

// Работа с одной задачей по ID (просмотр, обновление, удаление)
// handleItemByID godoc
// @Summary      Task by ID
// @Description  Get, update or delete single task
// @Tags         tasks
// @Accept       json
// @Produce      json
// @Param        id path int true "Task ID"
// @Param        data body TaskUpdateRequest true "Fields to update"
// @Success      200 {object} model.TaskDTO
// @Failure      400 {string} string "bad id"
// @Failure      404 {string} string "not found"
// @Router       /item/{id} [get]
// @Security     BearerAuth
// @Router       /item/{id} [put]
// @Security     BearerAuth
// @Router       /item/{id} [delete]
func (s *Server) handleItemByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/item/")
	idRaw := strings.TrimSpace(path)
	idNum, err := strconv.ParseInt(idRaw, 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	id := model.ID(idNum)

	switch r.Method {
	case http.MethodGet:
		for _, t := range s.svc.List(nil) {
			if t.ID() == id {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(t.ToDTO())
				return
			}
		}
		http.NotFound(w, r)

	case http.MethodPut:
		var dto TaskUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if dto.Title != "" {
			if err := s.svc.UpdateTitle(id, dto.Title); err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
		}
		if dto.Description != "" {
			s.svc.UpdateDesc(id, dto.Description)
		}
		if dto.Status != "" {
			s.svc.SetStatus(id, model.Status(dto.Status))
		}
		if dto.Priority > 0 {
			s.svc.SetPriority(id, model.Priority(dto.Priority))
		}
		if dto.DueAt != "" {
			if dto.DueAt == "-" {
				s.svc.ClearDue(id)
			} else if t, err := time.Parse("2006-01-02", dto.DueAt); err == nil {
				s.svc.SetDue(id, t)
			}
		}
		w.WriteHeader(http.StatusOK)

	case http.MethodDelete:
		if err := s.svc.Delete(id); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
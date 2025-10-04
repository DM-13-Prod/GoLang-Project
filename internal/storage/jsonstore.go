package storage

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"copybook/internal/model"
)

type JSONStore struct {
	Path string
}

func NewJSONStore(path string) *JSONStore {
	return &JSONStore{Path: path}
}

func (s *JSONStore) Load() ([]model.TaskDTO, error) {
	if s.Path == "" {
		return nil, errors.New("empty store path")
	}
	_ = os.MkdirAll(filepath.Dir(s.Path), 0o755)

	data, err := os.ReadFile(s.Path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []model.TaskDTO{}, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return []model.TaskDTO{}, nil
	}

	var items []model.TaskDTO
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *JSONStore) Save(items []model.TaskDTO) error {
	if s.Path == "" {
		return errors.New("empty store path")
	}
	_ = os.MkdirAll(filepath.Dir(s.Path), 0o755)

	raw, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}

	tmp := s.Path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.Path)
}
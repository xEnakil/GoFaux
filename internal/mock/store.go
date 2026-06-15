package mock

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

type Store struct {
	mu       sync.RWMutex
	filePath string
	data     Set
}

func DefaultConfigPath() string {
	return "gofaux.mocks.json"
}

func NewMemoryStore() *Store {
	return &Store{
		data: Set{
			Version: ConfigVersion,
			Name:    "GoFaux local mocks",
			APIs:    []Definition{},
		},
	}
}

func NewStore(filePath string) (*Store, error) {
	if filePath == "" {
		filePath = DefaultConfigPath()
	}

	store := NewMemoryStore()
	store.filePath = filePath
	if err := store.Load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) ConfigPath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.filePath
}

func (s *Store) Load() error {
	if s.filePath == "" {
		return nil
	}

	content, err := os.ReadFile(s.filePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read mock config: %w", err)
	}
	if len(content) == 0 {
		return nil
	}

	var loaded Set
	if err := json.Unmarshal(content, &loaded); err != nil {
		return fmt.Errorf("parse mock config: %w", err)
	}
	if loaded.Version == 0 {
		loaded.Version = ConfigVersion
	}
	if loaded.Name == "" {
		loaded.Name = "GoFaux local mocks"
	}
	for i := range loaded.APIs {
		if err := loaded.APIs[i].Normalize(); err != nil {
			return fmt.Errorf("mock %d is invalid: %w", i+1, err)
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = loaded
	return nil
}

func (s *Store) Save() error {
	s.mu.RLock()
	snapshot := s.data
	snapshot.APIs = CloneDefinitions(snapshot.APIs)
	filePath := s.filePath
	s.mu.RUnlock()

	return writeSet(filePath, snapshot)
}

func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data.APIs)
}

func (s *Store) All() []Definition {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return CloneDefinitions(s.data.APIs)
}

func (s *Store) Add(input Definition) (Definition, error) {
	def, err := NewDefinition(input)
	if err != nil {
		return Definition{}, err
	}

	s.mu.Lock()
	s.data.APIs = append(s.data.APIs, def)
	s.mu.Unlock()

	if err := s.Save(); err != nil {
		return Definition{}, err
	}
	return def, nil
}

func (s *Store) DeleteByIndex(index int) (Definition, bool, error) {
	s.mu.Lock()
	if index < 0 || index >= len(s.data.APIs) {
		s.mu.Unlock()
		return Definition{}, false, nil
	}

	removed := s.data.APIs[index]
	s.data.APIs = append(s.data.APIs[:index], s.data.APIs[index+1:]...)
	s.mu.Unlock()

	if err := s.Save(); err != nil {
		return Definition{}, false, err
	}
	return CloneDefinition(removed), true, nil
}

func (s *Store) DeleteByID(id string) (Definition, bool, error) {
	s.mu.Lock()
	for i := range s.data.APIs {
		if s.data.APIs[i].ID != id {
			continue
		}
		removed := s.data.APIs[i]
		s.data.APIs = append(s.data.APIs[:i], s.data.APIs[i+1:]...)
		s.mu.Unlock()

		if err := s.Save(); err != nil {
			return Definition{}, false, err
		}
		return CloneDefinition(removed), true, nil
	}
	s.mu.Unlock()
	return Definition{}, false, nil
}

func (s *Store) Find(r *http.Request) (Definition, Match, bool) {
	defs := s.All()
	SortByMatchPriority(defs)
	for _, def := range defs {
		match, ok := def.Matches(r)
		if ok {
			return def, match, true
		}
	}
	return Definition{}, Match{}, false
}

func (s *Store) RecordRequest(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.data.APIs {
		if s.data.APIs[i].ID == id {
			s.data.APIs[i].Requests++
			return
		}
	}
}

func writeSet(filePath string, set Set) error {
	if filePath == "" {
		return nil
	}
	if set.Version == 0 {
		set.Version = ConfigVersion
	}
	if set.Name == "" {
		set.Name = "GoFaux local mocks"
	}

	dir := filepath.Dir(filePath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create config directory: %w", err)
		}
	}

	content, err := json.MarshalIndent(set, "", "  ")
	if err != nil {
		return fmt.Errorf("serialize mock config: %w", err)
	}
	content = append(content, '\n')

	if err := os.WriteFile(filePath, content, 0o644); err != nil {
		return fmt.Errorf("write mock config: %w", err)
	}
	return nil
}

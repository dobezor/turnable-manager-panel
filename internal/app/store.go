package app

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
	"time"
)

type Store struct {
	path  string
	mu    sync.RWMutex
	state State
}

func OpenStore(path string) (*Store, string, error) {
	if err := EnsureParent(path, 0700); err != nil {
		return nil, "", err
	}
	raw, err := os.ReadFile(path)
	if err == nil {
		var st State
		if err := json.Unmarshal(raw, &st); err != nil {
			return nil, "", err
		}
		normalizeState(&st)
		return &Store{path: path, state: st}, "", nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, "", err
	}

	password := os.Getenv("TURNABLE_MANAGER_ADMIN_PASSWORD")
	generated := ""
	if strings.TrimSpace(password) == "" {
		generated = RandomHex(12)
		password = generated
	}
	hash, err := HashPassword(password)
	if err != nil {
		return nil, "", err
	}
	st := DefaultState(hash)
	store := &Store{path: path, state: st}
	if err := store.Save(); err != nil {
		return nil, "", err
	}
	return store, generated, nil
}

func normalizeState(st *State) {
	if st.Version == 0 {
		st.Version = 1
	}
	if st.AdminUsername == "" {
		st.AdminUsername = "admin"
	}
	if st.Settings.PlatformID == "" {
		st.Settings.PlatformID = "vk.com"
	}
	if st.Settings.RelayProto == "" {
		st.Settings.RelayProto = "dtls"
	}
	if st.Settings.RelayCloak == "" {
		st.Settings.RelayCloak = "none"
	}
	if st.Settings.RelayPort == 0 {
		st.Settings.RelayPort = 56000
	}
	if st.Settings.P2PCloak == "" {
		st.Settings.P2PCloak = "none"
	}
	if st.Settings.WorkDir == "" {
		st.Settings.WorkDir = "/etc/turnable"
	}
	if st.Settings.TurnableBinary == "" {
		st.Settings.TurnableBinary = "/usr/local/bin/turnable"
	}
	if st.Settings.TurnableService == "" {
		st.Settings.TurnableService = "turnable"
	}
	if st.Settings.XrayService == "" {
		st.Settings.XrayService = "xray"
	}
}

func (s *Store) Snapshot() State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := s.state
	cp.Routes = append([]Route(nil), s.state.Routes...)
	cp.Users = append([]User(nil), s.state.Users...)
	return cp
}

func (s *Store) Mutate(fn func(*State) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := fn(&s.state); err != nil {
		return err
	}
	s.state.UpdatedAt = time.Now().UTC()
	return s.saveLocked()
}

func (s *Store) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveLocked()
}

func (s *Store) saveLocked() error {
	raw, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *Store) FindUser(uuid string) (User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.state.Users {
		if u.UUID == uuid {
			return u, true
		}
	}
	return User{}, false
}

func (s *Store) FindRoute(id string) (Route, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, r := range s.state.Routes {
		if r.ID == id {
			return r, true
		}
	}
	return Route{}, false
}

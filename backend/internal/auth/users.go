package auth

import (
	"errors"
	"strings"
)

// Swap this with your real storage (DB, meta store, etc.)
type User struct {
	Username   string
	Email      string
	PassHash   string // argon2id encoded string
	Roles      []Role
	TOTPSecret string
}

type UserStore interface {
	FindByUsername(username string) (*User, error)
	FindByEmail(email string) (*User, error)
	Add(u *User) error
	UpdatePassword(username, newHash string) error
}

type MemoryUserStore struct {
	byUsername map[string]*User
	byEmail    map[string]*User
}

func NewMemoryUserStore() *MemoryUserStore {
	return &MemoryUserStore{
		byUsername: map[string]*User{},
		byEmail:    map[string]*User{},
	}
}

func (s *MemoryUserStore) Add(u *User) error {
	if u == nil {
		return errors.New("user is nil")
	}
	if s.byUsername == nil {
		s.byUsername = map[string]*User{}
	}
	if s.byEmail == nil {
		s.byEmail = map[string]*User{}
	}
	if _, exists := s.byUsername[u.Username]; exists {
		return errors.New("user already exists")
	}
	email := strings.ToLower(strings.TrimSpace(u.Email))
	if email != "" {
		if _, exists := s.byEmail[email]; exists {
			return errors.New("email already exists")
		}
	}
	clone := *u
	clone.Email = email
	s.byUsername[u.Username] = &clone
	if email != "" {
		s.byEmail[email] = &clone
	}
	return nil
}

func (s *MemoryUserStore) UpdatePassword(username, newHash string) error {
	if s.byUsername == nil {
		return errors.New("store not initialized")
	}
	u, ok := s.byUsername[username]
	if !ok {
		return errors.New("user not found")
	}
	u.PassHash = newHash
	return nil
}

func (s *MemoryUserStore) FindByUsername(username string) (*User, error) {
	if s.byUsername == nil {
		return nil, errors.New("store not initialized")
	}
	if u, ok := s.byUsername[username]; ok {
		clone := *u
		return &clone, nil
	}
	return nil, errors.New("user not found")
}

func (s *MemoryUserStore) FindByEmail(email string) (*User, error) {
	if s.byEmail == nil {
		return nil, errors.New("store not initialized")
	}
	if u, ok := s.byEmail[strings.ToLower(strings.TrimSpace(email))]; ok {
		clone := *u
		return &clone, nil
	}
	return nil, errors.New("user not found")
}

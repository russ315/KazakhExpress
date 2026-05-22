package user

import (
	"context"
	"errors"
	"sync"
	"time"
)

type mockRepository struct {
	mu           sync.RWMutex
	users        map[string]*User
	tokens       map[string]*RefreshToken
	blacklist    map[string]time.Time
	resetTokens  map[string]resetTokenInfo
	createErr    error
	getByIDErr   error
	getByEmailErr error
	updateErr    error
}

type resetTokenInfo struct {
	tokenHash string
	expiresAt time.Time
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		users:       make(map[string]*User),
		tokens:      make(map[string]*RefreshToken),
		blacklist:   make(map[string]time.Time),
		resetTokens: make(map[string]resetTokenInfo),
	}
}

func (m *mockRepository) Create(user *User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.createErr != nil {
		return m.createErr
	}
	if _, exists := m.users[user.ID]; exists {
		return errors.New("user already exists")
	}
	for _, u := range m.users {
		if u.Email == user.Email {
			return errors.New("email already exists")
		}
	}
	uCopy := *user
	m.users[user.ID] = &uCopy
	return nil
}

func (m *mockRepository) GetByID(id string) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.getByIDErr != nil {
		return nil, m.getByIDErr
	}
	user, exists := m.users[id]
	if !exists {
		return nil, errors.New("user not found")
	}
	// return a copy
	c := *user
	return &c, nil
}

func (m *mockRepository) GetByEmail(email string) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.getByEmailErr != nil {
		return nil, m.getByEmailErr
	}
	for _, u := range m.users {
		if u.Email == email {
			c := *u
			return &c, nil
		}
	}
	return nil, errors.New("user not found")
}

func (m *mockRepository) Update(user *User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.updateErr != nil {
		return m.updateErr
	}
	if _, exists := m.users[user.ID]; !exists {
		return errors.New("user not found")
	}
	uCopy := *user
	m.users[user.ID] = &uCopy
	return nil
}

func (m *mockRepository) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.users, id)
	return nil
}

func (m *mockRepository) SaveRefreshToken(token *RefreshToken) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokens[token.TokenHash] = token
	return nil
}

func (m *mockRepository) GetRefreshToken(tokenHash string) (*RefreshToken, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, exists := m.tokens[tokenHash]
	if !exists {
		return nil, errors.New("refresh token not found")
	}
	return t, nil
}

func (m *mockRepository) DeleteRefreshToken(tokenHash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.tokens, tokenHash)
	return nil
}

func (m *mockRepository) DeleteUserRefreshTokens(userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, v := range m.tokens {
		if v.UserID == userID {
			delete(m.tokens, k)
		}
	}
	return nil
}

func (m *mockRepository) AddToBlacklist(jti string, expiresAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.blacklist[jti] = expiresAt
	return nil
}

func (m *mockRepository) IsBlacklisted(jti string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	exp, exists := m.blacklist[jti]
	if !exists {
		return false, nil
	}
	if time.Now().After(exp) {
		return false, nil
	}
	return true, nil
}

func (m *mockRepository) SavePasswordResetToken(userID string, tokenHash string, expiresAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.resetTokens[userID] = resetTokenInfo{tokenHash: tokenHash, expiresAt: expiresAt}
	return nil
}

func (m *mockRepository) GetUserByResetToken(tokenHash string) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for uid, info := range m.resetTokens {
		if info.tokenHash == tokenHash {
			if time.Now().After(info.expiresAt) {
				return nil, errors.New("expired reset token")
			}
			user, exists := m.users[uid]
			if exists {
				c := *user
				return &c, nil
			}
		}
	}
	return nil, errors.New("invalid or expired reset token")
}

func (m *mockRepository) UpdatePassword(userID string, passwordHash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, exists := m.users[userID]
	if !exists {
		return errors.New("user not found")
	}
	u.Password = passwordHash
	return nil
}

func (m *mockRepository) ClearPasswordResetToken(userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.resetTokens, userID)
	return nil
}

// mockEmailService
type mockEmailService struct {
	mu           sync.Mutex
	welcomeSent  map[string]string // email -> firstName
	sendErr      error
}

func newMockEmailService() *mockEmailService {
	return &mockEmailService{
		welcomeSent: make(map[string]string),
	}
}

func (m *mockEmailService) SendWelcomeEmail(email, firstName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sendErr != nil {
		return m.sendErr
	}
	m.welcomeSent[email] = firstName
	return nil
}

// mockEventService
type mockEventService struct {
	mu     sync.Mutex
	events []interface{}
}

func newMockEventService() *mockEventService {
	return &mockEventService{}
}

func (m *mockEventService) PublishUserEvent(ctx context.Context, event interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
	return nil
}

// mockCacheService
type mockCacheService struct {
	mu         sync.Mutex
	users      map[string]interface{}
	blacklist  map[string]time.Duration
	invalidated []string
	getCachedErr error
}

func newMockCacheService() *mockCacheService {
	return &mockCacheService{
		users:     make(map[string]interface{}),
		blacklist: make(map[string]time.Duration),
	}
}

func (m *mockCacheService) GetCachedUser(ctx context.Context, userID string, dest interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.getCachedErr != nil {
		return m.getCachedErr
	}
	u, exists := m.users[userID]
	if !exists {
		return errors.New("cache miss")
	}
	if userDest, ok := dest.(*User); ok {
		if srcUser, ok := u.(*User); ok {
			*userDest = *srcUser
			return nil
		}
	}
	return nil
}

func (m *mockCacheService) CacheUser(ctx context.Context, userID string, data interface{}, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[userID] = data
	return nil
}

func (m *mockCacheService) InvalidateUserCache(ctx context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.invalidated = append(m.invalidated, userID)
	delete(m.users, userID)
	return nil
}

func (m *mockCacheService) BlacklistToken(ctx context.Context, jti string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.blacklist[jti] = ttl
	return nil
}

func (m *mockCacheService) IsTokenBlacklisted(ctx context.Context, jti string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, exists := m.blacklist[jti]
	return exists, nil
}

// mockRateLimitService
type mockRateLimitService struct {
	mu       sync.Mutex
	attempts map[string]int
}

func newMockRateLimitService() *mockRateLimitService {
	return &mockRateLimitService{
		attempts: make(map[string]int),
	}
}

func (m *mockRateLimitService) CheckLoginRateLimit(ctx context.Context, identifier string, maxAttempts int, window time.Duration) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.attempts[identifier]++
	return m.attempts[identifier], nil
}

func (m *mockRateLimitService) ResetLoginRateLimit(ctx context.Context, identifier string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.attempts, identifier)
	return nil
}

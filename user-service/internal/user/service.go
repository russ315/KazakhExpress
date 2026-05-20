package user

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Service interface {
	Register(input *RegisterInput) (*AuthResponse, error)
	Login(input *LoginInput) (*AuthResponse, error)
	GetProfile(userID string) (*User, error)
	UpdateProfile(userID string, input *UpdateProfileInput) (*User, error)
	ValidateToken(tokenString string) (string, error)
	RefreshToken(refreshTokenStr string) (*AuthResponse, error)
	Logout(userID, accessToken, refreshTokenStr string) error
	ForgotPassword(input *ForgotPasswordInput) error
	ResetPassword(input *ResetPasswordInput) error
}

type UserService struct {
	repo        Repository
	jwtSecret   string
	emailSvc    EmailService
	eventSvc    EventService
	cacheSvc    CacheService
	rateLimitSvc RateLimitService
}

type EmailService interface {
	SendWelcomeEmail(email, firstName string) error
}

type EventService interface {
	PublishUserEvent(ctx context.Context, event interface{}) error
}

type CacheService interface {
	GetCachedUser(ctx context.Context, userID string, dest interface{}) error
	CacheUser(ctx context.Context, userID string, data interface{}, ttl time.Duration) error
	InvalidateUserCache(ctx context.Context, userID string) error
	BlacklistToken(ctx context.Context, jti string, ttl time.Duration) error
	IsTokenBlacklisted(ctx context.Context, jti string) (bool, error)
}

type RateLimitService interface {
	CheckLoginRateLimit(ctx context.Context, identifier string, maxAttempts int, window time.Duration) (int, error)
	ResetLoginRateLimit(ctx context.Context, identifier string) error
}

func NewService(repo Repository, jwtSecret string, emailSvc EmailService, eventSvc EventService, cacheSvc CacheService, rateLimitSvc RateLimitService) *UserService {
	return &UserService{
		repo:          repo,
		jwtSecret:     jwtSecret,
		emailSvc:      emailSvc,
		eventSvc:      eventSvc,
		cacheSvc:      cacheSvc,
		rateLimitSvc:  rateLimitSvc,
	}
}

func (s *UserService) Register(input *RegisterInput) (*AuthResponse, error) {
	existingUser, err := s.repo.GetByEmail(input.Email)
	if err == nil && existingUser != nil {
		return nil, fmt.Errorf("user with email %s already exists", input.Email)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &User{
		ID:        uuid.New().String(),
		Email:     input.Email,
		Password:  string(hashedPassword),
		FirstName: input.FirstName,
		LastName:  input.LastName,
		Phone:     input.Phone,
		Address:   input.Address,
	}

	if err := s.repo.Create(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	token, refreshTokenStr, err := s.generateTokenPair(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	if s.emailSvc != nil {
		go func() {
			if err := s.emailSvc.SendWelcomeEmail(user.Email, user.FirstName); err != nil {
				log.Printf("Failed to send welcome email: %v", err)
			}
		}()
	}

	if s.eventSvc != nil {
		go s.publishEvent(context.Background(), EventUserCreated, user)
	}

	user.Password = ""

	return &AuthResponse{
		User:         *user,
		Token:        token,
		RefreshToken: refreshTokenStr,
	}, nil
}

func (s *UserService) Login(input *LoginInput) (*AuthResponse, error) {
	user, err := s.repo.GetByEmail(input.Email)
	if err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	if s.rateLimitSvc != nil {
		attempts, err := s.rateLimitSvc.CheckLoginRateLimit(context.Background(), input.Email, 5, 15*time.Minute)
		if err != nil {
			log.Printf("Rate limit check error: %v", err)
		} else if attempts > 5 {
			return nil, fmt.Errorf("too many login attempts, please try again later")
		}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	if s.rateLimitSvc != nil {
		s.rateLimitSvc.ResetLoginRateLimit(context.Background(), input.Email)
	}

	token, refreshTokenStr, err := s.generateTokenPair(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	user.Password = ""

	return &AuthResponse{
		User:         *user,
		Token:        token,
		RefreshToken: refreshTokenStr,
	}, nil
}

func (s *UserService) GetProfile(userID string) (*User, error) {
	if s.cacheSvc != nil {
		var cached User
		if err := s.cacheSvc.GetCachedUser(context.Background(), userID, &cached); err == nil {
			cached.Password = ""
			return &cached, nil
		}
	}

	user, err := s.repo.GetByID(userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	if s.cacheSvc != nil {
		go s.cacheSvc.CacheUser(context.Background(), userID, user, 5*time.Minute)
	}

	user.Password = ""
	return user, nil
}

func (s *UserService) UpdateProfile(userID string, input *UpdateProfileInput) (*User, error) {
	user, err := s.repo.GetByID(userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	if input.FirstName != nil {
		user.FirstName = *input.FirstName
	}
	if input.LastName != nil {
		user.LastName = *input.LastName
	}
	if input.Phone != nil {
		user.Phone = *input.Phone
	}
	if input.Address != nil {
		user.Address = *input.Address
	}

	if err := s.repo.Update(user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	if s.cacheSvc != nil {
		go s.cacheSvc.InvalidateUserCache(context.Background(), userID)
	}

	if s.eventSvc != nil {
		go s.publishEvent(context.Background(), EventUserUpdated, user)
	}

	user.Password = ""
	return user, nil
}

func (s *UserService) ValidateToken(tokenString string) (string, error) {
	if s.cacheSvc != nil {
		blacklisted, err := s.cacheSvc.IsTokenBlacklisted(context.Background(), tokenString)
		if err == nil && blacklisted {
			return "", fmt.Errorf("token has been revoked")
		}
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		return "", fmt.Errorf("invalid token: %w", err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userID, ok := claims["user_id"].(string)
		if !ok {
			return "", fmt.Errorf("invalid user ID in token")
		}
		return userID, nil
	}

	return "", fmt.Errorf("invalid token")
}

func (s *UserService) RefreshToken(refreshTokenStr string) (*AuthResponse, error) {
	tokenHash := hashToken(refreshTokenStr)

	savedToken, err := s.repo.GetRefreshToken(tokenHash)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token")
	}

	if time.Now().After(savedToken.ExpiresAt) {
		s.repo.DeleteRefreshToken(tokenHash)
		return nil, fmt.Errorf("refresh token expired")
	}

	user, err := s.repo.GetByID(savedToken.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	s.repo.DeleteRefreshToken(tokenHash)

	token, newRefreshTokenStr, err := s.generateTokenPair(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	user.Password = ""

	return &AuthResponse{
		User:         *user,
		Token:        token,
		RefreshToken: newRefreshTokenStr,
	}, nil
}

func (s *UserService) Logout(userID, accessToken, refreshTokenStr string) error {
	if s.cacheSvc != nil {
		go s.cacheSvc.BlacklistToken(context.Background(), accessToken, 7*24*time.Hour)
	}

	if refreshTokenStr != "" {
		tokenHash := hashToken(refreshTokenStr)
		s.repo.DeleteRefreshToken(tokenHash)
	}

	s.repo.DeleteUserRefreshTokens(userID)

	if s.cacheSvc != nil {
		go s.cacheSvc.InvalidateUserCache(context.Background(), userID)
	}

	return nil
}

func (s *UserService) ForgotPassword(input *ForgotPasswordInput) error {
	user, err := s.repo.GetByEmail(input.Email)
	if err != nil {
		return nil
	}

	resetToken := generateRandomToken()
	tokenHash := hashToken(resetToken)

	expiresAt := time.Now().Add(1 * time.Hour)
	if err := s.repo.SavePasswordResetToken(user.ID, tokenHash, expiresAt); err != nil {
		return fmt.Errorf("failed to save reset token: %w", err)
	}

	log.Printf("Password reset token for %s: %s", user.Email, resetToken)

	return nil
}

func (s *UserService) ResetPassword(input *ResetPasswordInput) error {
	tokenHash := hashToken(input.Token)

	user, err := s.repo.GetUserByResetToken(tokenHash)
	if err != nil {
		return fmt.Errorf("invalid or expired reset token")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	if err := s.repo.UpdatePassword(user.ID, string(hashedPassword)); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	if err := s.repo.ClearPasswordResetToken(user.ID); err != nil {
		log.Printf("Failed to clear reset token: %v", err)
	}

	s.repo.DeleteUserRefreshTokens(user.ID)

	if s.cacheSvc != nil {
		go s.cacheSvc.InvalidateUserCache(context.Background(), user.ID)
	}

	return nil
}

func (s *UserService) generateTokenPair(userID string) (string, string, error) {
	jti := uuid.New().String()

	accessClaims := jwt.MapClaims{
		"user_id": userID,
		"jti":     jti,
		"exp":     time.Now().Add(time.Hour * 24 * 7).Unix(),
		"iat":     time.Now().Unix(),
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	tokenString, err := accessToken.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", "", err
	}

	refreshTokenRaw := uuid.New().String() + "-" + userID
	refreshTokenHash := hashToken(refreshTokenRaw)

	refreshToken := &RefreshToken{
		UserID:    userID,
		TokenHash: refreshTokenHash,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}

	if err := s.repo.SaveRefreshToken(refreshToken); err != nil {
		return "", "", fmt.Errorf("failed to save refresh token: %w", err)
	}

	return tokenString, refreshTokenRaw, nil
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func generateRandomToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

type EventType string

const (
	EventUserCreated EventType = "user.created"
	EventUserUpdated EventType = "user.updated"
)

type UserEvent struct {
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Event     EventType `json:"event"`
	Timestamp time.Time `json:"timestamp"`
}

func (s *UserService) publishEvent(ctx context.Context, eventType EventType, user *User) {
	if s.eventSvc == nil {
		return
	}

	event := UserEvent{
		UserID:    user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Event:     eventType,
		Timestamp: time.Now(),
	}

	if err := s.eventSvc.PublishUserEvent(ctx, event); err != nil {
		log.Printf("Failed to publish %s event: %v", eventType, err)
	}
}

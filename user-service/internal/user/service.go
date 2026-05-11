package user

import (
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
}

type UserService struct {
	repo         Repository
	jwtSecret    string
	emailService EmailService
}

type EmailService interface {
	SendWelcomeEmail(email, firstName string) error
}

func NewService(repo Repository, jwtSecret string, emailService EmailService) *UserService {
	return &UserService{
		repo:         repo,
		jwtSecret:    jwtSecret,
		emailService: emailService,
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

	token, err := s.generateToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	if s.emailService != nil {
		if err := s.emailService.SendWelcomeEmail(user.Email, user.FirstName); err != nil {
			log.Printf("Failed to send welcome email: %v", err)
		}
	}

	user.Password = ""

	return &AuthResponse{
		User:  *user,
		Token: token,
	}, nil
}

func (s *UserService) Login(input *LoginInput) (*AuthResponse, error) {
	user, err := s.repo.GetByEmail(input.Email)
	if err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	token, err := s.generateToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	user.Password = ""

	return &AuthResponse{
		User:  *user,
		Token: token,
	}, nil
}

func (s *UserService) GetProfile(userID string) (*User, error) {
	user, err := s.repo.GetByID(userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
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

	user.Password = ""
	return user, nil
}

func (s *UserService) generateToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 24 * 7).Unix(), // 7 days
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

func (s *UserService) ValidateToken(tokenString string) (string, error) {
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

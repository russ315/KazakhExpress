package user

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

type Repository interface {
	Create(user *User) error
	GetByID(id string) (*User, error)
	GetByEmail(email string) (*User, error)
	Update(user *User) error
	Delete(id string) error
	SaveRefreshToken(token *RefreshToken) error
	GetRefreshToken(tokenHash string) (*RefreshToken, error)
	DeleteRefreshToken(tokenHash string) error
	DeleteUserRefreshTokens(userID string) error
	AddToBlacklist(jti string, expiresAt time.Time) error
	IsBlacklisted(jti string) (bool, error)
	SavePasswordResetToken(userID string, tokenHash string, expiresAt time.Time) error
	GetUserByResetToken(tokenHash string) (*User, error)
	UpdatePassword(userID string, passwordHash string) error
	ClearPasswordResetToken(userID string) error
}

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(dbURL string) (*PostgresRepository, error) {
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	repo := &PostgresRepository{db: db}
	if err := repo.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return repo, nil
}

func (r *PostgresRepository) createTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id VARCHAR(36) PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			first_name VARCHAR(100) NOT NULL,
			last_name VARCHAR(100) NOT NULL,
			phone VARCHAR(20),
			address TEXT,
			reset_token_hash VARCHAR(255),
			reset_token_expires_at TIMESTAMP WITH TIME ZONE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)`,

		`CREATE TABLE IF NOT EXISTS refresh_tokens (
			id SERIAL PRIMARY KEY,
			user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			token_hash VARCHAR(255) NOT NULL UNIQUE,
			expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token_hash ON refresh_tokens(token_hash)`,

		`CREATE TABLE IF NOT EXISTS token_blacklist (
			id SERIAL PRIMARY KEY,
			jti VARCHAR(255) NOT NULL UNIQUE,
			expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_token_blacklist_jti ON token_blacklist(jti)`,
		`CREATE INDEX IF NOT EXISTS idx_token_blacklist_expires_at ON token_blacklist(expires_at)`,
	}

	for _, q := range queries {
		if _, err := r.db.Exec(q); err != nil {
			return fmt.Errorf("failed to execute: %s: %w", q[:60], err)
		}
	}
	return nil
}

func (r *PostgresRepository) Create(user *User) error {
	query := `
	INSERT INTO users (id, email, password_hash, first_name, last_name, phone, address, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	_, err := r.db.Exec(query, user.ID, user.Email, user.Password, user.FirstName, user.LastName, user.Phone, user.Address, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		log.Printf("Error creating user: %v", err)
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetByID(id string) (*User, error) {
	query := `
	SELECT id, email, password_hash, first_name, last_name, phone, address, created_at, updated_at
	FROM users
	WHERE id = $1
	`

	user := &User{}
	err := r.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.Email,
		&user.Password,
		&user.FirstName,
		&user.LastName,
		&user.Phone,
		&user.Address,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

func (r *PostgresRepository) GetByEmail(email string) (*User, error) {
	query := `
	SELECT id, email, password_hash, first_name, last_name, phone, address, created_at, updated_at
	FROM users
	WHERE email = $1
	`

	user := &User{}
	err := r.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.Email,
		&user.Password,
		&user.FirstName,
		&user.LastName,
		&user.Phone,
		&user.Address,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

func (r *PostgresRepository) Update(user *User) error {
	query := `
	UPDATE users
	SET first_name = $2, last_name = $3, phone = $4, address = $5, updated_at = $6
	WHERE id = $1
	`

	user.UpdatedAt = time.Now()

	_, err := r.db.Exec(query, user.ID, user.FirstName, user.LastName, user.Phone, user.Address, user.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

func (r *PostgresRepository) Delete(id string) error {
	query := `DELETE FROM users WHERE id = $1`

	_, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

func (r *PostgresRepository) SaveRefreshToken(token *RefreshToken) error {
	query := `
	INSERT INTO refresh_tokens (user_id, token_hash, expires_at, created_at)
	VALUES ($1, $2, $3, $4)
	`

	token.CreatedAt = time.Now()

	_, err := r.db.Exec(query, token.UserID, token.TokenHash, token.ExpiresAt, token.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to save refresh token: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetRefreshToken(tokenHash string) (*RefreshToken, error) {
	query := `
	SELECT id, user_id, token_hash, expires_at, created_at
	FROM refresh_tokens
	WHERE token_hash = $1
	`

	token := &RefreshToken{}
	err := r.db.QueryRow(query, tokenHash).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("refresh token not found")
		}
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	return token, nil
}

func (r *PostgresRepository) DeleteRefreshToken(tokenHash string) error {
	query := `DELETE FROM refresh_tokens WHERE token_hash = $1`
	_, err := r.db.Exec(query, tokenHash)
	return err
}

func (r *PostgresRepository) DeleteUserRefreshTokens(userID string) error {
	query := `DELETE FROM refresh_tokens WHERE user_id = $1`
	_, err := r.db.Exec(query, userID)
	return err
}

func (r *PostgresRepository) AddToBlacklist(jti string, expiresAt time.Time) error {
	query := `
	INSERT INTO token_blacklist (jti, expires_at, created_at)
	VALUES ($1, $2, $3)
	ON CONFLICT (jti) DO NOTHING
	`

	_, err := r.db.Exec(query, jti, expiresAt, time.Now())
	return err
}

func (r *PostgresRepository) IsBlacklisted(jti string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM token_blacklist WHERE jti = $1)`

	var exists bool
	err := r.db.QueryRow(query, jti).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (r *PostgresRepository) SavePasswordResetToken(userID string, tokenHash string, expiresAt time.Time) error {
	query := `
	UPDATE users
	SET reset_token_hash = $2, reset_token_expires_at = $3, updated_at = NOW()
	WHERE id = $1
	`

	_, err := r.db.Exec(query, userID, tokenHash, expiresAt)
	return err
}

func (r *PostgresRepository) GetUserByResetToken(tokenHash string) (*User, error) {
	query := `
	SELECT id, email, password_hash, first_name, last_name, phone, address, created_at, updated_at
	FROM users
	WHERE reset_token_hash = $1 AND reset_token_expires_at > NOW()
	`

	user := &User{}
	err := r.db.QueryRow(query, tokenHash).Scan(
		&user.ID,
		&user.Email,
		&user.Password,
		&user.FirstName,
		&user.LastName,
		&user.Phone,
		&user.Address,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invalid or expired reset token")
		}
		return nil, fmt.Errorf("failed to get user by reset token: %w", err)
	}

	return user, nil
}

func (r *PostgresRepository) UpdatePassword(userID string, passwordHash string) error {
	query := `
	UPDATE users
	SET password_hash = $2, updated_at = NOW()
	WHERE id = $1
	`

	_, err := r.db.Exec(query, userID, passwordHash)
	return err
}

func (r *PostgresRepository) ClearPasswordResetToken(userID string) error {
	query := `
	UPDATE users
	SET reset_token_hash = NULL, reset_token_expires_at = NULL, updated_at = NOW()
	WHERE id = $1
	`

	_, err := r.db.Exec(query, userID)
	return err
}

func (r *PostgresRepository) Close() error {
	return r.db.Close()
}

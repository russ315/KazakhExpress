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
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id VARCHAR(36) PRIMARY KEY,
		email VARCHAR(255) UNIQUE NOT NULL,
		password_hash VARCHAR(255) NOT NULL,
		first_name VARCHAR(100) NOT NULL,
		last_name VARCHAR(100) NOT NULL,
		phone VARCHAR(20),
		address TEXT,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
	`

	_, err := r.db.Exec(query)
	return err
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

func (r *PostgresRepository) Close() error {
	return r.db.Close()
}

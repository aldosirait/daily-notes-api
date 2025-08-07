package repository

import (
	"database/sql"
	"fmt"

	"daily-notes-api/internal/models"
	"daily-notes-api/pkg/auth"
)

type UserRepository interface {
	Create(req *models.RegisterRequest) (*models.User, error)
	GetByUsername(username string) (*models.User, error)
	GetByEmail(email string) (*models.User, error)
	GetByID(id int) (*models.User, error)
	UpdateProfile(userID int, fullName, email string) (*models.User, error)
	ChangePassword(userID int, newPasswordHash string) error
	UsernameExists(username string) (bool, error)
	EmailExists(email string) (bool, error)
}

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(req *models.RegisterRequest) (*models.User, error) {
	// Hash password
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	query := `INSERT INTO users (username, email, password_hash, full_name) VALUES (?, ?, ?, ?)`
	result, err := r.db.Exec(query, req.Username, req.Email, passwordHash, req.FullName)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return r.GetByID(int(id))
}

func (r *userRepository) GetByUsername(username string) (*models.User, error) {
	query := `SELECT id, username, email, password_hash, full_name, created_at, updated_at 
              FROM users WHERE username = ?`

	user := &models.User{}
	err := r.db.QueryRow(query, username).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.FullName, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}

	return user, nil
}

func (r *userRepository) GetByEmail(email string) (*models.User, error) {
	query := `SELECT id, username, email, password_hash, full_name, created_at, updated_at 
              FROM users WHERE email = ?`

	user := &models.User{}
	err := r.db.QueryRow(query, email).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.FullName, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return user, nil
}

func (r *userRepository) GetByID(id int) (*models.User, error) {
	query := `SELECT id, username, email, password_hash, full_name, created_at, updated_at 
              FROM users WHERE id = ?`

	user := &models.User{}
	err := r.db.QueryRow(query, id).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.FullName, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	return user, nil
}

func (r *userRepository) UpdateProfile(userID int, fullName, email string) (*models.User, error) {
	query := `UPDATE users SET full_name = ?, email = ? WHERE id = ?`
	result, err := r.db.Exec(query, fullName, email, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to update user profile: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return nil, nil
	}

	return r.GetByID(userID)
}

func (r *userRepository) ChangePassword(userID int, newPasswordHash string) error {
	query := `UPDATE users SET password_hash = ? WHERE id = ?`
	result, err := r.db.Exec(query, newPasswordHash, userID)
	if err != nil {
		return fmt.Errorf("failed to change password: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

func (r *userRepository) UsernameExists(username string) (bool, error) {
	query := `SELECT COUNT(*) FROM users WHERE username = ?`
	var count int
	err := r.db.QueryRow(query, username).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check username existence: %w", err)
	}
	return count > 0, nil
}

func (r *userRepository) EmailExists(email string) (bool, error) {
	query := `SELECT COUNT(*) FROM users WHERE email = ?`
	var count int
	err := r.db.QueryRow(query, email).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check email existence: %w", err)
	}
	return count > 0, nil
}

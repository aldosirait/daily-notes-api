package repository

import (
	"database/sql"
	"fmt"
	"strings"

	"daily-notes-api/internal/models"
)

type NoteRepository interface {
	Create(userID int, note *models.CreateNoteRequest) (*models.Note, error)
	GetByID(id, userID int) (*models.Note, error)
	GetAll(userID int, filter *models.NotesFilter) ([]*models.Note, int, error)
	Update(id, userID int, note *models.UpdateNoteRequest) (*models.Note, error)
	Delete(id, userID int) error
	GetCategories(userID int) ([]string, error)
}

type noteRepository struct {
	db *sql.DB
}

func NewNoteRepository(db *sql.DB) NoteRepository {
	return &noteRepository{db: db}
}

func (r *noteRepository) Create(userID int, req *models.CreateNoteRequest) (*models.Note, error) {
	query := `INSERT INTO notes (user_id, title, content, category) VALUES (?, ?, ?, ?)`
	result, err := r.db.Exec(query, userID, req.Title, req.Content, req.Category)
	if err != nil {
		return nil, fmt.Errorf("failed to create note: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return r.GetByID(int(id), userID)
}

func (r *noteRepository) GetByID(id, userID int) (*models.Note, error) {
	query := `SELECT n.id, n.user_id, n.title, n.content, n.category, n.created_at, n.updated_at,
                     u.id, u.username, u.email, u.full_name, u.created_at
              FROM notes n
              LEFT JOIN users u ON n.user_id = u.id
              WHERE n.id = ? AND n.user_id = ?`

	note := &models.Note{}
	user := &models.UserResponse{}

	err := r.db.QueryRow(query, id, userID).Scan(
		&note.ID, &note.UserID, &note.Title, &note.Content, &note.Category,
		&note.CreatedAt, &note.UpdatedAt,
		&user.ID, &user.Username, &user.Email, &user.FullName, &user.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get note: %w", err)
	}

	note.User = user
	return note, nil
}

func (r *noteRepository) GetAll(userID int, filter *models.NotesFilter) ([]*models.Note, int, error) {
	var conditions []string
	var args []interface{}

	// Always filter by user_id
	conditions = append(conditions, "n.user_id = ?")
	args = append(args, userID)

	baseQuery := `FROM notes n LEFT JOIN users u ON n.user_id = u.id`

	if filter.Category != "" {
		conditions = append(conditions, "n.category = ?")
		args = append(args, filter.Category)
	}

	whereClause := " WHERE " + strings.Join(conditions, " AND ")

	// Get total count
	countQuery := "SELECT COUNT(*) " + baseQuery + whereClause
	var total int
	err := r.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get count: %w", err)
	}

	// Get notes with pagination
	offset := (filter.Page - 1) * filter.Limit
	selectQuery := `SELECT n.id, n.user_id, n.title, n.content, n.category, n.created_at, n.updated_at,
                           u.id, u.username, u.email, u.full_name, u.created_at ` +
		baseQuery + whereClause +
		` ORDER BY n.created_at DESC LIMIT ? OFFSET ?`

	args = append(args, filter.Limit, offset)
	rows, err := r.db.Query(selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get notes: %w", err)
	}
	defer rows.Close()

	var notes []*models.Note
	for rows.Next() {
		note := &models.Note{}
		user := &models.UserResponse{}

		err := rows.Scan(
			&note.ID, &note.UserID, &note.Title, &note.Content, &note.Category,
			&note.CreatedAt, &note.UpdatedAt,
			&user.ID, &user.Username, &user.Email, &user.FullName, &user.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan note: %w", err)
		}

		note.User = user
		notes = append(notes, note)
	}

	return notes, total, nil
}

func (r *noteRepository) Update(id, userID int, req *models.UpdateNoteRequest) (*models.Note, error) {
	query := `UPDATE notes SET title = ?, content = ?, category = ? WHERE id = ? AND user_id = ?`
	result, err := r.db.Exec(query, req.Title, req.Content, req.Category, id, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to update note: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return nil, nil // Note not found or not owned by user
	}

	return r.GetByID(id, userID)
}

func (r *noteRepository) Delete(id, userID int) error {
	query := `DELETE FROM notes WHERE id = ? AND user_id = ?`
	result, err := r.db.Exec(query, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete note: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("note not found or not owned by user")
	}

	return nil
}

func (r *noteRepository) GetCategories(userID int) ([]string, error) {
	query := `SELECT DISTINCT category FROM notes WHERE category != '' AND user_id = ? ORDER BY category`
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var category string
		if err := rows.Scan(&category); err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}
		categories = append(categories, category)
	}

	return categories, nil
}

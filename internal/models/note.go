package models

import (
	"time"
)

type Note struct {
	ID        int       `json:"id" db:"id"`
	UserID    int       `json:"user_id" db:"user_id"`
	Title     string    `json:"title" db:"title" binding:"required,min=1,max=255"`
	Content   string    `json:"content" db:"content" binding:"required,min=1"`
	Category  string    `json:"category" db:"category" binding:"max=100"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	// Optional: Include user info
	User *UserResponse `json:"user,omitempty"`
}

type CreateNoteRequest struct {
	Title    string `json:"title" binding:"required,min=1,max=255"`
	Content  string `json:"content" binding:"required,min=1"`
	Category string `json:"category" binding:"max=100"`
}

type UpdateNoteRequest struct {
	Title    string `json:"title" binding:"required,min=1,max=255"`
	Content  string `json:"content" binding:"required,min=1"`
	Category string `json:"category" binding:"max=100"`
}

type NotesFilter struct {
	Category string `form:"category"`
	Page     int    `form:"page"`  // Hapus binding:"min=1"
	Limit    int    `form:"limit"` // Hapus binding:"min=1,max=100"
}

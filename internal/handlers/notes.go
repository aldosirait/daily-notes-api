package handlers

import (
	"context"
	"log"
	"strconv"
	"time"

	"daily-notes-api/internal/middleware"
	"daily-notes-api/internal/models"
	"daily-notes-api/internal/repository"
	"daily-notes-api/pkg/cache"
	"daily-notes-api/pkg/response"

	"github.com/gin-gonic/gin"
)

type NoteHandler struct {
	noteRepo     repository.NoteRepository
	cacheService *cache.CacheService
}

func NewNoteHandler(noteRepo repository.NoteRepository, cacheService *cache.CacheService) *NoteHandler {
	return &NoteHandler{
		noteRepo:     noteRepo,
		cacheService: cacheService,
	}
}

func (h *NoteHandler) CreateNote(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req models.CreateNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationErrors(c, err)
		return
	}

	note, err := h.noteRepo.Create(userID, &req)
	if err != nil {
		response.InternalServerError(c, "Failed to create note")
		return
	}

	// Invalidate cache after creating a note
	h.invalidateCache(c, userID, 0)

	response.Created(c, note)
}

func (h *NoteHandler) GetNote(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.ValidationErrorResponse(c, response.ValidationError{
			Field:   "id",
			Message: "Note ID must be a valid number",
			Value:   idStr,
		})
		return
	}

	// Try to get from cache first
	var note *models.Note
	cacheKey := ""

	if h.cacheService != nil {
		cacheKey = h.cacheService.GenerateNoteDetailKey(id, userID)
		ctx, cancel := context.WithTimeout(c, 5*time.Second)
		defer cancel()

		err := h.cacheService.Get(ctx, cacheKey, &note)
		if err == nil && note != nil {
			log.Printf("Cache hit for note detail: %s", cacheKey)
			response.Success(c, note)
			return
		}

		if err != nil {
			log.Printf("Cache miss for note detail %s: %v", cacheKey, err)
		}
	}

	// Get from database
	note, err = h.noteRepo.GetByID(id, userID)
	if err != nil {
		response.InternalServerError(c, "Failed to get note")
		return
	}

	if note == nil {
		response.NotFound(c, "Note not found")
		return
	}

	// Cache the result
	if h.cacheService != nil && cacheKey != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := h.cacheService.Set(ctx, cacheKey, note, 30*time.Minute); err != nil {
			log.Printf("Failed to cache note detail: %v", err)
		}
	}

	response.Success(c, note)
}

func (h *NoteHandler) GetNotes(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var filter models.NotesFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		response.ValidationErrors(c, err)
		return
	}

	// Manual validation with better error messages
	var validationErrors []response.ValidationError

	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Limit <= 0 {
		filter.Limit = 10
	}
	if filter.Limit > 100 {
		validationErrors = append(validationErrors, response.ValidationError{
			Field:   "limit",
			Message: "Limit cannot exceed 100 items per page",
			Value:   strconv.Itoa(filter.Limit),
		})
	}

	if len(validationErrors) > 0 {
		response.ValidationErrorsResponse(c, validationErrors)
		return
	}

	// Try to get from cache first
	var notes []*models.Note
	var total int
	cacheKey := ""

	if h.cacheService != nil {
		cacheKey = h.cacheService.GenerateNotesListKey(userID, filter.Category, filter.Page, filter.Limit)
		ctx, cancel := context.WithTimeout(c, 5*time.Second)
		defer cancel()

		// Create a struct to cache both notes and total
		type CachedNotesData struct {
			Notes []*models.Note `json:"notes"`
			Total int            `json:"total"`
		}

		var cachedData CachedNotesData
		err := h.cacheService.Get(ctx, cacheKey, &cachedData)
		if err == nil {
			log.Printf("Cache hit for notes list: %s", cacheKey)
			meta := response.CalculatePagination(filter.Page, filter.Limit, cachedData.Total)
			if notes == nil {
				notes = make([]*models.Note, 0)
			}
			response.SuccessWithMeta(c, notes, meta)
			return
		}

		if err != nil {
			log.Printf("Cache miss for notes list %s: %v", cacheKey, err)
		}
	}

	// Get from database
	notes, total, err := h.noteRepo.GetAll(userID, &filter)
	if err != nil {
		response.InternalServerError(c, "Failed to get notes")
		return
	}

	// Cache the result
	if h.cacheService != nil && cacheKey != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		type CachedNotesData struct {
			Notes []*models.Note `json:"notes"`
			Total int            `json:"total"`
		}

		cachedData := CachedNotesData{
			Notes: notes,
			Total: total,
		}

		if err := h.cacheService.Set(ctx, cacheKey, cachedData, 30*time.Minute); err != nil {
			log.Printf("Failed to cache notes list: %v", err)
		}
	}

	meta := response.CalculatePagination(filter.Page, filter.Limit, total)
	response.SuccessWithMeta(c, notes, meta)
}

func (h *NoteHandler) UpdateNote(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.ValidationErrorResponse(c, response.ValidationError{
			Field:   "id",
			Message: "Note ID must be a valid number",
			Value:   idStr,
		})
		return
	}

	var req models.UpdateNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationErrors(c, err)
		return
	}

	note, err := h.noteRepo.Update(id, userID, &req)
	if err != nil {
		response.InternalServerError(c, "Failed to update note")
		return
	}

	if note == nil {
		response.NotFound(c, "Note not found or you don't have permission to update it")
		return
	}

	// Invalidate cache after updating a note
	h.invalidateCache(c, userID, id)

	response.Success(c, note)
}

func (h *NoteHandler) DeleteNote(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.ValidationErrorResponse(c, response.ValidationError{
			Field:   "id",
			Message: "Note ID must be a valid number",
			Value:   idStr,
		})
		return
	}

	if err := h.noteRepo.Delete(id, userID); err != nil {
		if err.Error() == "note not found or not owned by user" {
			response.NotFound(c, "Note not found or you don't have permission to delete it")
			return
		}
		response.InternalServerError(c, "Failed to delete note")
		return
	}

	// Invalidate cache after deleting a note
	h.invalidateCache(c, userID, id)

	response.Success(c, gin.H{"message": "Note deleted successfully"})
}

func (h *NoteHandler) GetCategories(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	// Try to get from cache first
	var categories []string
	cacheKey := ""

	if h.cacheService != nil {
		cacheKey = h.cacheService.GenerateCategoriesKey(userID)
		ctx, cancel := context.WithTimeout(c, 5*time.Second)
		defer cancel()

		err := h.cacheService.Get(ctx, cacheKey, &categories)
		if err == nil {
			log.Printf("Cache hit for categories: %s", cacheKey)
			response.Success(c, categories)
			return
		}

		if err != nil {
			log.Printf("Cache miss for categories %s: %v", cacheKey, err)
		}
	}

	// Get from database
	categories, err := h.noteRepo.GetCategories(userID)
	if err != nil {
		response.InternalServerError(c, "Failed to get categories")
		return
	}

	// Cache the result
	if h.cacheService != nil && cacheKey != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := h.cacheService.Set(ctx, cacheKey, categories, 30*time.Minute); err != nil {
			log.Printf("Failed to cache categories: %v", err)
		}
	}

	response.Success(c, categories)
}

// invalidateCache removes cache entries after note operations
func (h *NoteHandler) invalidateCache(c *gin.Context, userID, noteID int) {
	if h.cacheService == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if noteID > 0 {
		// Invalidate specific note cache
		if err := h.cacheService.InvalidateNoteCache(ctx, noteID, userID); err != nil {
			log.Printf("Failed to invalidate note cache: %v", err)
		}
	} else {
		// Invalidate all user notes cache (for create operations)
		if err := h.cacheService.InvalidateUserNotesCache(ctx, userID); err != nil {
			log.Printf("Failed to invalidate user notes cache: %v", err)
		}
	}

	log.Printf("Cache invalidated for user %d, note %d", userID, noteID)
}

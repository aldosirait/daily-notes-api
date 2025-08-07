package handlers

import (
	"strconv"

	"daily-notes-api/internal/middleware"
	"daily-notes-api/internal/models"
	"daily-notes-api/internal/repository"
	"daily-notes-api/pkg/response"

	"github.com/gin-gonic/gin"
)

type NoteHandler struct {
	noteRepo repository.NoteRepository
}

func NewNoteHandler(noteRepo repository.NoteRepository) *NoteHandler {
	return &NoteHandler{noteRepo: noteRepo}
}

func (h *NoteHandler) CreateNote(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req models.CreateNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	note, err := h.noteRepo.Create(userID, &req)
	if err != nil {
		response.InternalServerError(c, "Failed to create note")
		return
	}

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
		response.BadRequest(c, "Invalid note ID")
		return
	}

	note, err := h.noteRepo.GetByID(id, userID)
	if err != nil {
		response.InternalServerError(c, "Failed to get note")
		return
	}

	if note == nil {
		response.NotFound(c, "Note not found")
		return
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
		response.BadRequest(c, "Invalid query parameters: "+err.Error())
		return
	}

	// Manual validation dan set default
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Limit <= 0 {
		filter.Limit = 10
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}

	notes, total, err := h.noteRepo.GetAll(userID, &filter)
	if err != nil {
		response.InternalServerError(c, "Failed to get notes")
		return
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
		response.BadRequest(c, "Invalid note ID")
		return
	}

	var req models.UpdateNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body: "+err.Error())
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
		response.BadRequest(c, "Invalid note ID")
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

	response.Success(c, gin.H{"message": "Note deleted successfully"})
}

func (h *NoteHandler) GetCategories(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	categories, err := h.noteRepo.GetCategories(userID)
	if err != nil {
		response.InternalServerError(c, "Failed to get categories")
		return
	}

	response.Success(c, categories)
}

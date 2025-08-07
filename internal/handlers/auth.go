package handlers

import (
	"strings"

	"daily-notes-api/internal/models"
	"daily-notes-api/internal/repository"
	"daily-notes-api/pkg/auth"
	"daily-notes-api/pkg/response"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	userRepo   repository.UserRepository
	jwtManager *auth.JWTManager
}

func NewAuthHandler(userRepo repository.UserRepository, jwtManager *auth.JWTManager) *AuthHandler {
	return &AuthHandler{
		userRepo:   userRepo,
		jwtManager: jwtManager,
	}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationErrors(c, err)
		return
	}

	// Validate and clean input
	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)
	req.FullName = strings.TrimSpace(req.FullName)

	// Additional custom validation
	var validationErrors []response.ValidationError

	if len(req.Username) < 3 {
		validationErrors = append(validationErrors, response.ValidationError{
			Field:   "username",
			Message: "Username must be at least 3 characters long",
			Value:   req.Username,
		})
	}

	if len(req.Password) < 6 {
		validationErrors = append(validationErrors, response.ValidationError{
			Field:   "password",
			Message: "Password must be at least 6 characters long",
		})
	}

	if len(validationErrors) > 0 {
		response.ValidationErrorsResponse(c, validationErrors)
		return
	}

	// Check if username already exists
	exists, err := h.userRepo.UsernameExists(req.Username)
	if err != nil {
		response.InternalServerError(c, "Failed to check username availability")
		return
	}
	if exists {
		response.Conflict(c, "Username already exists")
		return
	}

	// Check if email already exists
	exists, err = h.userRepo.EmailExists(req.Email)
	if err != nil {
		response.InternalServerError(c, "Failed to check email availability")
		return
	}
	if exists {
		response.Conflict(c, "Email already exists")
		return
	}

	// Create user
	user, err := h.userRepo.Create(&req)
	if err != nil {
		response.InternalServerError(c, "Failed to create user")
		return
	}

	// Generate JWT token
	token, err := h.jwtManager.GenerateToken(user.ID, user.Username)
	if err != nil {
		response.InternalServerError(c, "Failed to generate token")
		return
	}

	loginResponse := &models.LoginResponse{
		User:  user.ToResponse(),
		Token: token,
	}

	response.Created(c, loginResponse)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationErrors(c, err)
		return
	}

	// Additional validation for empty fields
	if strings.TrimSpace(req.Username) == "" {
		response.ValidationErrorResponse(c, response.ValidationError{
			Field:   "username",
			Message: "Username is required and cannot be empty",
		})
		return
	}

	if strings.TrimSpace(req.Password) == "" {
		response.ValidationErrorResponse(c, response.ValidationError{
			Field:   "password",
			Message: "Password is required and cannot be empty",
		})
		return
	}

	// Get user by username
	user, err := h.userRepo.GetByUsername(req.Username)
	if err != nil {
		response.InternalServerError(c, "Failed to authenticate user")
		return
	}

	if user == nil {
		response.Unauthorized(c, "Invalid username or password")
		return
	}

	// Verify password
	if !auth.CheckPasswordHash(req.Password, user.PasswordHash) {
		response.Unauthorized(c, "Invalid username or password")
		return
	}

	// Generate JWT token
	token, err := h.jwtManager.GenerateToken(user.ID, user.Username)
	if err != nil {
		response.InternalServerError(c, "Failed to generate token")
		return
	}

	loginResponse := &models.LoginResponse{
		User:  user.ToResponse(),
		Token: token,
	}

	response.Success(c, loginResponse)
}

func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	user, err := h.userRepo.GetByID(userID.(int))
	if err != nil {
		response.InternalServerError(c, "Failed to get user profile")
		return
	}

	if user == nil {
		response.NotFound(c, "User not found")
		return
	}

	response.Success(c, user.ToResponse())
}

func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req struct {
		FullName string `json:"full_name" binding:"required,min=2,max=100"`
		Email    string `json:"email" binding:"required,email,max=100"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationErrors(c, err)
		return
	}

	// Check if email already exists (exclude current user)
	existingUser, err := h.userRepo.GetByEmail(req.Email)
	if err != nil {
		response.InternalServerError(c, "Failed to check email availability")
		return
	}
	if existingUser != nil && existingUser.ID != userID.(int) {
		response.Conflict(c, "Email already exists")
		return
	}

	user, err := h.userRepo.UpdateProfile(userID.(int), req.FullName, req.Email)
	if err != nil {
		response.InternalServerError(c, "Failed to update profile")
		return
	}

	if user == nil {
		response.NotFound(c, "User not found")
		return
	}

	response.Success(c, user.ToResponse())
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required,min=6,max=100"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationErrors(c, err)
		return
	}

	// Get current user
	user, err := h.userRepo.GetByID(userID.(int))
	if err != nil {
		response.InternalServerError(c, "Failed to get user")
		return
	}

	if user == nil {
		response.NotFound(c, "User not found")
		return
	}

	// Verify current password
	if !auth.CheckPasswordHash(req.CurrentPassword, user.PasswordHash) {
		response.Unauthorized(c, "Current password is incorrect")
		return
	}

	// Hash new password
	newPasswordHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		response.InternalServerError(c, "Failed to hash new password")
		return
	}

	// Update password
	err = h.userRepo.ChangePassword(userID.(int), newPasswordHash)
	if err != nil {
		response.InternalServerError(c, "Failed to change password")
		return
	}

	response.Success(c, gin.H{"message": "Password changed successfully"})
}

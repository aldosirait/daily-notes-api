package response

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	Meta    any    `json:"meta,omitempty"`
	Errors  any    `json:"errors,omitempty"`
}

type PaginationMeta struct {
	Page      int `json:"page"`
	Limit     int `json:"limit"`
	Total     int `json:"total"`
	TotalPage int `json:"total_page"`
}

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   string `json:"value,omitempty"`
}

// Error implements the error interface
func (ve ValidationError) Error() string {
	return fmt.Sprintf("validation error in field '%s': %s", ve.Field, ve.Message)
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
	})
}

func SuccessWithMeta(c *gin.Context, data any, meta any) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Response{
		Success: true,
		Data:    data,
	})
}

func BadRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, Response{
		Success: false,
		Message: message,
	})
}

// ValidationErrors handles validation errors with HTTP 422
func ValidationErrors(c *gin.Context, err error) {
	var validationErrors []ValidationError

	switch e := err.(type) {
	case validator.ValidationErrors:
		// Handle struct validation errors
		for _, fieldErr := range e {
			validationError := ValidationError{
				Field:   getFieldName(fieldErr),
				Message: getValidationMessage(fieldErr),
				Value:   fmt.Sprintf("%v", fieldErr.Value()),
			}
			validationErrors = append(validationErrors, validationError)
		}
	case *json.UnmarshalTypeError:
		// Handle JSON unmarshaling errors (like your case)
		fieldName := strings.ToLower(e.Field)
		var message string

		switch e.Type.Kind() {
		case reflect.String:
			message = fmt.Sprintf("Field must be a string, received %s", e.Type.Name())
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			message = fmt.Sprintf("Field must be a number, received %s", e.Type.Name())
		case reflect.Bool:
			message = fmt.Sprintf("Field must be a boolean, received %s", e.Type.Name())
		case reflect.Slice:
			message = fmt.Sprintf("Field must be an array, received %s", e.Type.Name())
		default:
			message = fmt.Sprintf("Invalid data type for field, expected %s", e.Type.Name())
		}

		validationError := ValidationError{
			Field:   fieldName,
			Message: message,
			Value:   fmt.Sprintf("%v", e.Value),
		}
		validationErrors = append(validationErrors, validationError)
	case *json.SyntaxError:
		// Handle JSON syntax errors
		c.JSON(http.StatusUnprocessableEntity, Response{
			Success: false,
			Message: "Invalid JSON format",
			Errors:  "Please check your JSON syntax",
		})
		return
	default:
		// Handle other validation-related errors
		if strings.Contains(err.Error(), "json:") {
			// Try to parse JSON-related errors
			errorMsg := err.Error()
			if strings.Contains(errorMsg, "cannot unmarshal") {
				// Extract field name and type from error message
				parts := strings.Split(errorMsg, " ")
				var field, expectedType string

				for i, part := range parts {
					if part == "field" && i+1 < len(parts) {
						fieldParts := strings.Split(parts[i+1], ".")
						if len(fieldParts) > 1 {
							field = strings.ToLower(fieldParts[len(fieldParts)-1])
						}
					}
					if part == "type" && i+1 < len(parts) {
						expectedType = parts[i+1]
					}
				}

				if field == "" {
					// Try to extract from the full error message
					if strings.Contains(errorMsg, "CreateNoteRequest.") {
						start := strings.Index(errorMsg, "CreateNoteRequest.") + len("CreateNoteRequest.")
						end := strings.Index(errorMsg[start:], " ")
						if end != -1 {
							field = errorMsg[start : start+end]
						}
					}
				}

				message := "Invalid data type"
				switch expectedType {
				case "string":
					message = "This field must be a text value (string), not a number"
				case "number":
					message = "This field must be a number, not text"
				}

				validationError := ValidationError{
					Field:   field,
					Message: message,
				}
				validationErrors = append(validationErrors, validationError)
			} else {
				// Generic JSON error
				validationErrors = append(validationErrors, ValidationError{
					Field:   "request_body",
					Message: "Invalid request format",
				})
			}
		} else {
			// Generic validation error
			validationErrors = append(validationErrors, ValidationError{
				Field:   "unknown",
				Message: err.Error(),
			})
		}
	}

	c.JSON(http.StatusUnprocessableEntity, Response{
		Success: false,
		Message: "Validation failed",
		Errors:  validationErrors,
	})
}

// ValidationErrorResponse handles single ValidationError structs
func ValidationErrorResponse(c *gin.Context, ve ValidationError) {
	c.JSON(http.StatusUnprocessableEntity, Response{
		Success: false,
		Message: "Validation failed",
		Errors:  []ValidationError{ve},
	})
}

// ValidationErrorsResponse handles multiple ValidationError structs
func ValidationErrorsResponse(c *gin.Context, errors []ValidationError) {
	c.JSON(http.StatusUnprocessableEntity, Response{
		Success: false,
		Message: "Validation failed",
		Errors:  errors,
	})
}

func Unauthorized(c *gin.Context, message string) {
	c.JSON(http.StatusUnauthorized, Response{
		Success: false,
		Message: message,
	})
}

func Forbidden(c *gin.Context, message string) {
	c.JSON(http.StatusForbidden, Response{
		Success: false,
		Message: message,
	})
}

func NotFound(c *gin.Context, message string) {
	c.JSON(http.StatusNotFound, Response{
		Success: false,
		Message: message,
	})
}

func Conflict(c *gin.Context, message string) {
	c.JSON(http.StatusConflict, Response{
		Success: false,
		Message: message,
	})
}

func InternalServerError(c *gin.Context, message string) {
	c.JSON(http.StatusInternalServerError, Response{
		Success: false,
		Message: message,
	})
}

func CalculatePagination(page, limit, total int) PaginationMeta {
	totalPage := max((total+limit-1)/limit, 1)

	return PaginationMeta{
		Page:      page,
		Limit:     limit,
		Total:     total,
		TotalPage: totalPage,
	}
}

// Helper functions
func getFieldName(fieldErr validator.FieldError) string {
	// Convert field name to snake_case for API consistency
	field := fieldErr.Field()
	return toSnakeCase(field)
}

func getValidationMessage(fieldErr validator.FieldError) string {
	switch fieldErr.Tag() {
	case "required":
		return "This field is required"
	case "min":
		if fieldErr.Kind() == reflect.String {
			return fmt.Sprintf("This field must be at least %s characters long", fieldErr.Param())
		}
		return fmt.Sprintf("This field must be at least %s", fieldErr.Param())
	case "max":
		if fieldErr.Kind() == reflect.String {
			return fmt.Sprintf("This field must not exceed %s characters", fieldErr.Param())
		}
		return fmt.Sprintf("This field must not exceed %s", fieldErr.Param())
	case "email":
		return "This field must be a valid email address"
	case "len":
		return fmt.Sprintf("This field must be exactly %s characters long", fieldErr.Param())
	case "numeric":
		return "This field must contain only numbers"
	case "alpha":
		return "This field must contain only letters"
	case "alphanum":
		return "This field must contain only letters and numbers"
	case "url":
		return "This field must be a valid URL"
	default:
		return fmt.Sprintf("This field failed validation (%s)", fieldErr.Tag())
	}
}

func toSnakeCase(str string) string {
	var result strings.Builder
	for i, char := range str {
		if i > 0 && 'A' <= char && char <= 'Z' {
			result.WriteRune('_')
		}
		if 'A' <= char && char <= 'Z' {
			result.WriteRune(char - 'A' + 'a')
		} else {
			result.WriteRune(char)
		}
	}
	return result.String()
}

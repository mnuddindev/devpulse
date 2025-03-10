package utils

import "github.com/gofiber/fiber/v2"

// Response holds a standardized API response fields.
type Response struct {
	Code    int         `json:"code"`
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// Success sends a standardized success response.
func Success(c *fiber.Ctx, code int, message string, data interface{}) error {
	return c.Status(code).JSON(Response{
		Status:  "success",
		Code:    code,
		Message: message,
		Data:    data,
	})
}

// Error sends a standardized error response using CustomError.
func Error(c *fiber.Ctx, err *CustomError) error {
	return c.Status(err.Status).JSON(Response{
		Status:  "error",
		Code:    err.Status,
		Message: err.Message,
		Data:    nil,
	})
}

// PaginationResponse handles paginated data (for: post lists)
type PaginationResponse struct {
	Items      interface{} `json:"items"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PerPage    int         `json:"per_page"`
	TotalPages int         `json:"total_pages"`
}

func PaginatedSuccess(c *fiber.Ctx, code int, message string, items interface{}, total int64, page, perpage int) error {
	totalPages := int((total + int64(perpage) - 1) / int64(perpage))
	return c.Status(code).JSON(Response{
		Status:  "success",
		Code:    code,
		Message: message,
		Data: PaginationResponse{
			Items:      items,
			Total:      total,
			Page:       page,
			PerPage:    perpage,
			TotalPages: totalPages,
		},
	})
}

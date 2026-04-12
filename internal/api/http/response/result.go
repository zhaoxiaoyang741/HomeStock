package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Result is the standard envelope for API responses.
type Result[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

// Page is the standard list container embedded in Result.Data.
type Page[T any] struct {
	Items    []T `json:"items"`
	Total    int `json:"total"`
	Page     int `json:"page,omitempty"`
	PageSize int `json:"page_size,omitempty"`
}

func OK[T any](c *gin.Context, data T) { c.JSON(http.StatusOK, Result[T]{Code: 0, Message: "ok", Data: data}) }

func Created[T any](c *gin.Context, data T) {
	c.JSON(http.StatusCreated, Result[T]{Code: 0, Message: "ok", Data: data})
}

func NoContent(c *gin.Context) { c.Status(http.StatusNoContent) }

func List[T any](c *gin.Context, items []T, total int, page, pageSize int) {
	OK(c, Page[T]{Items: items, Total: total, Page: page, PageSize: pageSize})
}

func Error(c *gin.Context, httpStatus int, message string) {
	c.JSON(httpStatus, Result[any]{Code: httpStatus, Message: message})
}

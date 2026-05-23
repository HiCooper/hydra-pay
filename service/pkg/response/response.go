package response

import (
	"math"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Success    bool        `json:"success"`
	Data       interface{} `json:"data,omitempty"`
	Error      *ErrorObj   `json:"error,omitempty"`
	Pagination *Pagination `json:"pagination,omitempty"`
}

type ErrorObj struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

type Pagination struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{Success: true, Data: data})
}

func SuccessWithPagination(c *gin.Context, data interface{}, page, pageSize, total int) {
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
		Pagination: &Pagination{
			Page: page, PageSize: pageSize, Total: total, TotalPages: totalPages,
		},
	})
}

func Error(c *gin.Context, statusCode int, code, message string) {
	c.JSON(statusCode, Response{
		Success: false,
		Error:   &ErrorObj{Code: code, Message: message},
	})
}

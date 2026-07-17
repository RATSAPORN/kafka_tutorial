package errors

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type CommonResponse struct {
	StatusCode int    `json:"-"`
	Code       string `json:"code,omitempty"`
	Message    string `json:"message,omitempty"`
	Data       any    `json:"data,omitempty"`
}

func CommonSuccessResponse(c *gin.Context, data any) {
	c.JSON(http.StatusOK, CommonResponse{
		StatusCode: http.StatusOK,
		Code:       "0000",
		Message:    "ok",
		Data:       data,
	})
}

func CommonErrorResponse(c *gin.Context, resp *CommonResponse) {
	c.JSON(resp.StatusCode, resp)
}

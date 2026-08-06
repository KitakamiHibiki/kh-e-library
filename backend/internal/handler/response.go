package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response is the unified API response format.
type Response struct {
	Code int         `json:"code"`
	Data interface{} `json:"data"`
	Msg  string      `json:"msg"`
}

// Success sends a 200 OK response with data.
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{Code: 200, Data: data, Msg: "ok"})
}

// SuccessWithStatus sends a response with a custom HTTP status code.
func SuccessWithStatus(c *gin.Context, httpStatus int, data interface{}) {
	c.JSON(httpStatus, Response{Code: httpStatus, Data: data, Msg: "ok"})
}

// Error sends an error response with the given HTTP status code and message.
func Error(c *gin.Context, httpStatus int, msg string) {
	c.JSON(httpStatus, Response{Code: httpStatus, Data: nil, Msg: msg})
}

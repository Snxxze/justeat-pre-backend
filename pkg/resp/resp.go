package resp

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"ok": true, "data": data})
}
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, gin.H{"ok": true, "data": data})
}
func BadRequest(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": msg})
}
func Unauthorized(c *gin.Context, msg string) {
	c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": msg})
}
func Forbidden(c *gin.Context, msg string) {
	c.JSON(http.StatusForbidden, gin.H{"ok": false, "error": msg})
}
func ServerError(c *gin.Context, err error) {
	c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
}
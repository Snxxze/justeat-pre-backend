package utils

import "github.com/gin-gonic/gin"

func CurrentUserID(c *gin.Context) uint {
	v, _ := c.Get("userId")
	switch id := v.(type) {
	case uint:
		return id
	case int:
		return uint(id)
	case int64:
		return uint(id)
	case float64:
		return uint(id)
	default:
		return 0
	}
}

func CurrentRole(c *gin.Context) string {
	if v, ok := c.Get("role"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

package controllers

import (
	"backend/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type MenuOptionController struct {
	Service *services.MenuOptionService
}

func NewMenuOptionController(s *services.MenuOptionService) *MenuOptionController {
	return &MenuOptionController{Service: s}
}

// POST /owner/menus/:id/options
// POST /owner/menus/:id/options
func (ctl *MenuOptionController) AttachOption(c *gin.Context) {
    menuID, err := strconv.Atoi(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid menu id"})
        return
    }

    var body struct {
        OptionID uint `json:"optionId"`
    }
    if err := c.ShouldBindJSON(&body); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    if err := ctl.Service.Attach(uint(menuID), body.OptionID); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "option attached"})
}


// DELETE /owner/menus/:id/options/:optionId
func (ctl *MenuOptionController) DetachOption(c *gin.Context) {
	menuID, _ := strconv.Atoi(c.Param("id"))
	optID, _ := strconv.Atoi(c.Param("optionId"))

	if err := ctl.Service.Detach(uint(menuID), uint(optID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "option detached"})
}

// GET /menus/:id/options (public → ลูกค้าเรียกดู)
func (ctl *MenuOptionController) ListByMenu(c *gin.Context) {
	menuID, _ := strconv.Atoi(c.Param("id"))

	opts, err := ctl.Service.GetByMenu(uint(menuID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": opts})
}

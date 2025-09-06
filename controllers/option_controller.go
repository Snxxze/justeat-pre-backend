package controllers

import (
	"backend/entity"
	"backend/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type OptionController struct {
	Service *services.OptionService
}

func NewOptionController(s *services.OptionService) *OptionController {
	return &OptionController{Service: s}
}

// GET /options
func (ctl *OptionController) List(c *gin.Context) {
	opts, err := ctl.Service.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": opts})
}

// GET /options/:id
func (ctl *OptionController) Get(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	opt, err := ctl.Service.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "option not found"})
		return
	}
	c.JSON(http.StatusOK, opt)
}

// POST /owner/options
func (ctl *OptionController) Create(c *gin.Context) {
	var req entity.Option
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := ctl.Service.Create(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, req)
}

// PATCH /owner/options/:id
func (ctl *OptionController) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req entity.Option
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = uint(id)

	if err := ctl.Service.Update(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, req)
}

// DELETE /owner/options/:id
func (ctl *OptionController) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := ctl.Service.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "option deleted"})
}

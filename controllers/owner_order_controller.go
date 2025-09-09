// controllers/owner_order_controller.go
package controllers

import (
	"backend/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type OwnerOrderController struct {
	Svc *services.OrderService
}

func NewOwnerOrderController(s *services.OrderService) *OwnerOrderController {
	return &OwnerOrderController{Svc: s}
}

// GET /owner/restaurants/:id/orders
func (ctl *OwnerOrderController) List(c *gin.Context) {
	userID := c.GetUint("userId")

	restID64, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	restID := uint(restID64)

	var statusIDPtr *uint
	if s := c.Query("statusId"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			tmp := uint(v)
			statusIDPtr = &tmp
		}
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	out, err := ctl.Svc.ListForRestaurant(userID, restID, statusIDPtr, page, limit)
	if err != nil {
		if err.Error() == "forbidden" {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}

// GET /owner/restaurants/:id/orders/:orderId
func (ctl *OwnerOrderController) Detail(c *gin.Context) {
	userID := c.GetUint("userId")

	restID64, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	restID := uint(restID64)

	orderID64, _ := strconv.ParseUint(c.Param("orderId"), 10, 64)
	orderID := uint(orderID64)

	out, err := ctl.Svc.DetailForRestaurant(userID, restID, orderID)
	if err != nil {
		if err.Error() == "forbidden" {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}
	c.JSON(http.StatusOK, out)
}

func (ctl *OwnerOrderController) Accept(c *gin.Context) {
	userID := c.GetUint("userId")
	oid, _ := strconv.ParseUint(c.Param("orderId"), 10, 64)
	if err := ctl.Svc.OwnerAccept(userID, uint(oid)); err != nil { writeOrderErr(c, err); return }
	c.Status(http.StatusNoContent)
}
func (ctl *OwnerOrderController) Handoff(c *gin.Context) {
	userID := c.GetUint("userId")
	oid, _ := strconv.ParseUint(c.Param("orderId"), 10, 64)
	if err := ctl.Svc.OwnerHandoffToRider(userID, uint(oid)); err != nil { writeOrderErr(c, err); return }
	c.Status(http.StatusNoContent)
}
func (ctl *OwnerOrderController) Complete(c *gin.Context) {
	userID := c.GetUint("userId")
	oid, _ := strconv.ParseUint(c.Param("orderId"), 10, 64)
	if err := ctl.Svc.OwnerComplete(userID, uint(oid)); err != nil { writeOrderErr(c, err); return }
	c.Status(http.StatusNoContent)
}
func (ctl *OwnerOrderController) Cancel(c *gin.Context) {
	userID := c.GetUint("userId")
	oid, _ := strconv.ParseUint(c.Param("orderId"), 10, 64)
	if err := ctl.Svc.OwnerCancel(userID, uint(oid)); err != nil { writeOrderErr(c, err); return }
	c.Status(http.StatusNoContent)
}

func writeOrderErr(c *gin.Context, err error) {
	switch err.Error() {
	case "forbidden":
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
	case "invalid_or_conflict":
		c.JSON(http.StatusConflict, gin.H{"error": "invalid state or already updated"})
	default:
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	}
}
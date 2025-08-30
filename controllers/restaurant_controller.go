package controllers

import (
	"errors"
	"strconv"
	"time"

	"backend/entity"
	"backend/pkg/resp"
	"backend/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type RestaurantController struct{ DB *gorm.DB }
func NewRestaurantController(db *gorm.DB) *RestaurantController { return &RestaurantController{DB: db} }

// ===== User side =====

// GET /restaurants?q=&categoryId=&statusId=&page=&limit=
func (rc *RestaurantController) List(c *gin.Context) {
	q := c.Query("q")
	categoryID, _ := strconv.Atoi(c.DefaultQuery("categoryId", "0"))
	statusID, _ := strconv.Atoi(c.DefaultQuery("statusId", "0"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 { page = 1 }
	if limit <= 0 || limit > 100 { limit = 20 }
	offset := (page - 1) * limit

	dbq := rc.DB.Model(&entity.Restaurant{})
	if q != "" {
		dbq = dbq.Where("name LIKE ? OR address LIKE ?", "%"+q+"%", "%"+q+"%")
	}
	if categoryID > 0 { dbq = dbq.Where("restaurant_category_id = ?", categoryID) }
	if statusID > 0 { dbq = dbq.Where("restaurant_status_id = ?", statusID) }

	var total int64
	if err := dbq.Count(&total).Error; err != nil { resp.ServerError(c, err); return }

	var items []entity.Restaurant
	if err := dbq.
		Select("id, name, picture, restaurant_status_id").
		Order("id DESC").Limit(limit).Offset(offset).
		Find(&items).Error; err != nil {
		resp.ServerError(c, err); return
	}

	resp.OK(c, gin.H{"items": items, "page": page, "limit": limit, "total": total})
}

// GET /restaurants/:id
func (rc *RestaurantController) Detail(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil { resp.BadRequest(c, "invalid id"); return }

	var r entity.Restaurant
	if err := rc.DB.Select("id, name, address, description, picture, restaurant_status_id").
		First(&r, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { c.JSON(404, gin.H{"ok": false, "error": "not found"}); return }
		resp.ServerError(c, err); return
	}

	// แนบเมนูแค่บางส่วน (status = Available)
	var menus []entity.Menu
	if err := rc.DB.Model(&entity.Menu{}).
		Select("id, menu_name, price, picture, restaurant_id").
		Where("restaurant_id = ? AND menu_status_id = ?", r.ID, 1).
		Order("id DESC").Limit(20).
		Find(&menus).Error; err != nil {
		resp.ServerError(c, err); return
	}

	resp.OK(c, gin.H{
		"id": r.ID, "name": r.Name, "address": r.Address, "description": r.Description,
		"picture": r.Picture, "restaurantStatusId": r.RestaurantStatusID,
		"menus": menus,
	})
}

// ===== Partner /restaurant (owner/admin) =====

// helper: ตรวจว่า user เป็นเจ้าของร้านนี้หรือ admin
func (rc *RestaurantController) ensureOwnerOrAdmin(c *gin.Context, restaurantID uint) (*entity.Restaurant, bool) {
	role := utils.CurrentRole(c)
	userID := utils.CurrentUserID(c)

	var r entity.Restaurant
	if err := rc.DB.Select("id, user_id, name").First(&r, restaurantID).Error; err != nil {
		resp.BadRequest(c, "restaurant not found"); return nil, false
	}
	if role != "admin" && r.UserID != userID {
		resp.Forbidden(c, "not owner"); return nil, false
	}
	return &r, true
}

// GET /partner/restaurant/orders?restaurantId=&statusId=&page=&limit=
func (rc *RestaurantController) Orders(c *gin.Context) {
	restID64, _ := strconv.ParseUint(c.DefaultQuery("restaurantId", "0"), 10, 64)
	if restID64 == 0 { resp.BadRequest(c, "restaurantId required"); return }
	if _, ok := rc.ensureOwnerOrAdmin(c, uint(restID64)); !ok { return }

	statusID, _ := strconv.Atoi(c.DefaultQuery("statusId", "0"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 { page = 1 }
	if limit <= 0 || limit > 100 { limit = 20 }
	offset := (page - 1) * limit

	dbq := rc.DB.Model(&entity.Order{}).Where("restaurant_id = ?", uint(restID64))
	if statusID > 0 { dbq = dbq.Where("order_status_id = ?", statusID) }

	var total int64
	if err := dbq.Count(&total).Error; err != nil { resp.ServerError(c, err); return }

	type row struct {
		ID uint `json:"id"`
		Total int64 `json:"total"`
		OrderStatusID uint `json:"orderStatusId"`
		UserID uint `json:"userId"`
		CreatedAt time.Time `json:"createdAt"`
	}
	var items []row
	if err := dbq.Select("id, total, order_status_id, user_id, created_at").
		Order("id DESC").Limit(limit).Offset(offset).
		Find(&items).Error; err != nil {
		resp.ServerError(c, err); return
	}
	resp.OK(c, gin.H{"items": items, "page": page, "limit": limit, "total": total})
}

// GET /partner/restaurant/menu?restaurantId=&page=&limit=
func (rc *RestaurantController) Menus(c *gin.Context) {
	restID64, _ := strconv.ParseUint(c.DefaultQuery("restaurantId", "0"), 10, 64)
	if restID64 == 0 { resp.BadRequest(c, "restaurantId required"); return }
	if _, ok := rc.ensureOwnerOrAdmin(c, uint(restID64)); !ok { return }

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if page < 1 { page = 1 }
	if limit <= 0 || limit > 100 { limit = 50 }
	offset := (page - 1) * limit

	var total int64
	if err := rc.DB.Model(&entity.Menu{}).
		Where("restaurant_id = ?", uint(restID64)).
		Count(&total).Error; err != nil { resp.ServerError(c, err); return }

	var items []entity.Menu
	if err := rc.DB.Model(&entity.Menu{}).
		Select("id, menu_name, price, picture, menu_status_id, menu_type_id").
		Where("restaurant_id = ?", uint(restID64)).
		Order("id DESC").Limit(limit).Offset(offset).
		Find(&items).Error; err != nil {
		resp.ServerError(c, err); return
	}
	resp.OK(c, gin.H{"items": items, "page": page, "limit": limit, "total": total})
}

type CreateMenuReq struct {
	RestaurantID   uint   `json:"restaurantId" binding:"required"`
	MenuName       string `json:"menuName" binding:"required"`
	Detail         string `json:"detail"`
	Price          int64  `json:"price" binding:"required"`
	Picture        string `json:"picture"`
	MenuTypeID     uint   `json:"menuTypeId" binding:"required"`
	MenuStatusID   uint   `json:"menuStatusId" binding:"required"`
}

// POST /partner/restaurant/menu
func (rc *RestaurantController) CreateMenu(c *gin.Context) {
	var req CreateMenuReq
	if err := c.ShouldBindJSON(&req); err != nil { resp.BadRequest(c, err.Error()); return }
	if _, ok := rc.ensureOwnerOrAdmin(c, req.RestaurantID); !ok { return }

	m := entity.Menu{
		MenuName: req.MenuName, Detail: req.Detail, Price: req.Price, Picture: req.Picture,
		MenuTypeID: req.MenuTypeID, MenuStatusID: req.MenuStatusID, RestaurantID: req.RestaurantID,
	}
	if err := rc.DB.Create(&m).Error; err != nil { resp.ServerError(c, err); return }
	resp.Created(c, gin.H{"id": m.ID})
}

type UpdateMenuReq struct {
	ID            uint    `json:"id" binding:"required"`
	RestaurantID  uint    `json:"restaurantId" binding:"required"`
	MenuName      *string `json:"menuName"`
	Detail        *string `json:"detail"`
	Price         *int64  `json:"price"`
	Picture       *string `json:"picture"`
	MenuTypeID    *uint   `json:"menuTypeId"`
	MenuStatusID  *uint   `json:"menuStatusId"`
}

// PATCH /partner/restaurant/menu/:id
func (rc *RestaurantController) UpdateMenu(c *gin.Context) {
	menuID64, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var req UpdateMenuReq
	if err := c.ShouldBindJSON(&req); err != nil { resp.BadRequest(c, err.Error()); return }
	if req.ID == 0 { req.ID = uint(menuID64) }

	var m entity.Menu
	if err := rc.DB.Select("id, restaurant_id").First(&m, req.ID).Error; err != nil {
		resp.BadRequest(c, "menu not found"); return
	}
	if req.RestaurantID == 0 { req.RestaurantID = m.RestaurantID }
	if _, ok := rc.ensureOwnerOrAdmin(c, req.RestaurantID); !ok { return }

	updates := map[string]any{}
	if req.MenuName != nil { updates["menu_name"] = *req.MenuName }
	if req.Detail != nil { updates["detail"] = *req.Detail }
	if req.Price != nil { updates["price"] = *req.Price }
	if req.Picture != nil { updates["picture"] = *req.Picture }
	if req.MenuTypeID != nil { updates["menu_type_id"] = *req.MenuTypeID }
	if req.MenuStatusID != nil { updates["menu_status_id"] = *req.MenuStatusID }

	if len(updates) == 0 { resp.OK(c, gin.H{"updated": 0}); return }
	if err := rc.DB.Model(&m).Updates(updates).Error; err != nil { resp.ServerError(c, err); return }
	resp.OK(c, gin.H{"id": m.ID, "updated": 1})
}

// GET /partner/restaurant/account?restaurantId=
func (rc *RestaurantController) Account(c *gin.Context) {
	restID64, _ := strconv.ParseUint(c.DefaultQuery("restaurantId", "0"), 10, 64)
	if restID64 == 0 { resp.BadRequest(c, "restaurantId required"); return }
	r, ok := rc.ensureOwnerOrAdmin(c, uint(restID64)); if !ok { return }

	resp.OK(c, gin.H{"id": r.ID, "name": r.Name})
}

// GET /partner/restaurant/dashboard?restaurantId=
func (rc *RestaurantController) Dashboard(c *gin.Context) {
	restID64, _ := strconv.ParseUint(c.DefaultQuery("restaurantId", "0"), 10, 64)
	if restID64 == 0 { resp.BadRequest(c, "restaurantId required"); return }
	if _, ok := rc.ensureOwnerOrAdmin(c, uint(restID64)); !ok { return }

	start := time.Now().Truncate(24 * time.Hour)
	var ordersToday int64
	var revenue int64
	rc.DB.Model(&entity.Order{}).
		Where("restaurant_id = ? AND created_at >= ?", uint(restID64), start).
		Count(&ordersToday)
	rc.DB.Model(&entity.Order{}).
		Select("COALESCE(SUM(total),0)").
		Where("restaurant_id = ? AND created_at >= ? AND order_status_id = ?", uint(restID64), start, 4). // 4 = Completed
		Scan(&revenue)

	resp.OK(c, gin.H{"ordersToday": ordersToday, "revenue": revenue})
}

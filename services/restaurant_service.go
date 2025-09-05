package services

import (
	"backend/entity"
	"backend/repository"
	"backend/utils"
	"errors"
	"time"
)

type RestaurantService struct {
	Repo *repository.RestaurantRepository
}

func NewRestaurantService(repo *repository.RestaurantRepository) *RestaurantService {
	return &RestaurantService{Repo: repo}
}

// Create restaurant
func (s *RestaurantService) CreateRestaurant(req *entity.Restaurant, base64Img string) (uint, error) {
	if base64Img != "" {
		path, err := utils.SaveBase64Image(base64Img, "uploads/restaurants")
		if err != nil {
			return 0, err
		}
		req.Picture = path
	}

	if err := s.Repo.CreateRestaurant(req); err != nil {
		return 0, err
	}
	return req.ID, nil
}

// Menu list
func (s *RestaurantService) GetMenus(restaurantID uint, page, limit int) ([]entity.Menu, int64, error) {
	if page < 1 { page = 1 }
	if limit <= 0 || limit > 100 { limit = 50 }
	offset := (page - 1) * limit

	total, err := s.Repo.CountMenus(restaurantID)
	if err != nil {
		return nil, 0, err
	}
	items, err := s.Repo.FindMenus(restaurantID, limit, offset)
	return items, total, err
}

// Create menu with base64 image
func (s *RestaurantService) CreateMenu(req *entity.Menu, base64Img string) (uint, error) {
	if base64Img != "" {
		path, err := utils.SaveBase64Image(base64Img, "uploads/menus")
		if err != nil {
			return 0, err
		}
		req.Picture = path
	}
	if err := s.Repo.CreateMenu(req); err != nil {
		return 0, err
	}
	return req.ID, nil
}

// Update menu
func (s *RestaurantService) UpdateMenu(reqID uint, updates map[string]any, base64Img *string) error {
	m, err := s.Repo.FindMenuByID(reqID)
	if err != nil {
		return errors.New("menu not found")
	}
	if base64Img != nil && *base64Img != "" {
		path, err := utils.SaveBase64Image(*base64Img, "uploads/menus")
		if err != nil {
			return err
		}
		updates["picture"] = path
	}
	return s.Repo.UpdateMenu(m, updates)
}

// Dashboard
func (s *RestaurantService) Dashboard(restaurantID uint) (int64, int64, error) {
	start := time.Now().Truncate(24 * time.Hour).Format("2006-01-02 15:04:05")
	orders, err := s.Repo.CountOrdersToday(restaurantID, start)
	if err != nil {
		return 0, 0, err
	}
	revenue, err := s.Repo.SumRevenueToday(restaurantID, start)
	if err != nil {
		return 0, 0, err
	}
	return orders, revenue, nil
}

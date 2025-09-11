// services/restaurant_service.go
package services

import (
	"backend/entity"
	"backend/repository"
)

type RestaurantService struct {
	Repo *repository.RestaurantRepository
}

func NewRestaurantService(repo *repository.RestaurantRepository) *RestaurantService {
	return &RestaurantService{Repo: repo}
}

// ดึงร้านทั้งหมด
func (s *RestaurantService) List() ([]entity.Restaurant, error) {
	return s.Repo.FindAll()
}

// ดึงร้านตาม ID
func (s *RestaurantService) Get(id uint) (*entity.Restaurant, error) {
	return s.Repo.FindByID(id)
}

// อัปเดตร้าน
func (s *RestaurantService) Update(rest *entity.Restaurant) error {
	return s.Repo.Update(rest)
}

// ListFiltered ดึงร้านตาม categoryId และ statusId (optional)
func (s *RestaurantService) ListFiltered(categoryId, statusId string) ([]entity.Restaurant, error) {
	q := s.Repo.DB

	if categoryId != "" {
		q = q.Where("restaurant_category_id = ?", categoryId)
	}
	if statusId != "" {
		q = q.Where("restaurant_status_id = ?", statusId)
	}

	var rests []entity.Restaurant
	err := q.
		Preload("RestaurantCategory").
		Preload("RestaurantStatus").
		Preload("User").
		Find(&rests).Error
	return rests, err
}

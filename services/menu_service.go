// services/menu_service.go
package services

import (
	"backend/entity"
	"backend/repository"
)

type MenuService struct {
	Repo *repository.MenuRepository
}

func NewMenuService(repo *repository.MenuRepository) *MenuService {
	return &MenuService{Repo: repo}
}

func (s *MenuService) ListByRestaurant(restID uint) ([]entity.Menu, error) {
	return s.Repo.FindByRestaurant(restID)
}

func (s *MenuService) Get(id uint) (*entity.Menu, error) {
	return s.Repo.FindByID(id)
}

func (s *MenuService) Create(menu *entity.Menu) error {
	return s.Repo.Create(menu)
}

func (s *MenuService) Update(menu *entity.Menu) error {
	return s.Repo.Update(menu)
}

func (s *MenuService) Delete(id uint) error {
	return s.Repo.Delete(id)
}

func (s *MenuService) UpdateStatus(id uint, statusID uint) error {
    return s.Repo.UpdateStatus(id, statusID)
}

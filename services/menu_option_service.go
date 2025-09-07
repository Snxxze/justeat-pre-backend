package services

import (
	"backend/entity"
	"backend/repository"
)

type MenuOptionService struct {
	Repo *repository.MenuOptionRepository
}

func NewMenuOptionService(repo *repository.MenuOptionRepository) *MenuOptionService {
	return &MenuOptionService{Repo: repo}
}

func (s *MenuOptionService) Attach(menuID, optionID uint) error {
    return s.Repo.Attach(menuID, optionID)
}

func (s *MenuOptionService) Detach(menuID, optionID uint) error {
	return s.Repo.Detach(menuID, optionID)
}

func (s *MenuOptionService) GetByMenu(menuID uint) ([]entity.Option, error) {
	return s.Repo.FindByMenu(menuID)
}

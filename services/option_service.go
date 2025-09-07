package services

import (
	"backend/entity"
	"backend/repository"
)

type OptionService struct {
	Repo *repository.OptionRepository
}

func NewOptionService(repo *repository.OptionRepository) *OptionService {
	return &OptionService{Repo: repo}
}

func (s *OptionService) GetAll() ([]entity.Option, error) {
	return s.Repo.FindAll()
}

func (s *OptionService) GetByID(id uint) (*entity.Option, error) {
	return s.Repo.FindByID(id)
}

func (s *OptionService) Create(opt *entity.Option) error {
	return s.Repo.Create(opt)
}

func (s *OptionService) Update(opt *entity.Option) error {
	return s.Repo.Update(opt)
}

func (s *OptionService) Delete(id uint) error {
	return s.Repo.Delete(id)
}

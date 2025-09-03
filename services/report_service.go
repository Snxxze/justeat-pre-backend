package services

import (
    "backend/entity"
    "backend/repository"
    "time"
)

type ReportService struct {
    repo *repository.ReportRepository
}

func NewReportService(repo *repository.ReportRepository) *ReportService {
    return &ReportService{repo: repo}
}

func (s *ReportService) CreateReport(userID uint, name, email, phone, description, picture string, issueTypeID uint) (*entity.Report, error) {
    now := time.Now()
    report := &entity.Report{
        Name:        name,
        Email:       email,
        PhoneNumber: phone,
        Description: description,
        Picture:     picture,
        IssueTypeID: issueTypeID,
        UserID:      userID,
        DateAt:      &now,
    }
    if err := s.repo.Create(report); err != nil {
        return nil, err
    }
    return report, nil
}

func (s *ReportService) FindAllByUser(userID uint, out *[]entity.Report) error {
	return s.repo.FindAllByUser(userID, out)
}

func (s *ReportService) FindByIDAndUser(userID uint, reportID string) (*entity.Report, error) {
	return s.repo.FindByIDAndUser(userID, reportID)
}
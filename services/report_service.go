package services

import (
    "backend/entity"
    "backend/repository"
    "time"
    "strconv"
    "fmt"
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
        Status:      "pending", // ✅ เพิ่มตรงนี้
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

func (s *ReportService) FindAllReports(out *[]entity.Report) error {
    return s.repo.FindAll(out)
}

// ✅ เพิ่มฟังก์ชันใหม่สำหรับอัปเดตสถานะ
func (s *ReportService) UpdateStatus(reportID string, newStatus string) error {
    id, err := strconv.Atoi(reportID)
    if err != nil {
        return fmt.Errorf("invalid report ID: %s", reportID)
    }

    // (ทางเลือก) ตรวจสอบสถานะที่อนุญาต
    validStatuses := map[string]bool{
        "pending":      true,
        "in_progress":  true,
        "resolved":     true,
        "closed":       true,
    }

    if !validStatuses[newStatus] {
        return fmt.Errorf("invalid status: %s", newStatus)
    }

    return s.repo.UpdateStatus(uint(id), newStatus)
}

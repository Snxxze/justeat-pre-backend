package repository

import (
    "backend/entity"
    "gorm.io/gorm"
)

type ReportRepository struct {
    db *gorm.DB
}

func NewReportRepository(db *gorm.DB) *ReportRepository {
    return &ReportRepository{db: db}
}

func (r *ReportRepository) Create(report *entity.Report) error {
    return r.db.Create(report).Error
}

func (r *ReportRepository) FindByID(id uint) (*entity.Report, error) {
    var report entity.Report
    if err := r.db.First(&report, id).Error; err != nil {
        return nil, err
    }
    return &report, nil
}

func (r *ReportRepository) FindAllByUser(userID uint, out *[]entity.Report) error {
	return r.db.Where("user_id = ?", userID).Find(out).Error
}

func (r *ReportRepository) FindByIDAndUser(userID uint, reportID string) (*entity.Report, error) {
	var report entity.Report
	if err := r.db.Where("id = ? AND user_id = ?", reportID, userID).First(&report).Error; err != nil {
		return nil, err
	}
	return &report, nil
}
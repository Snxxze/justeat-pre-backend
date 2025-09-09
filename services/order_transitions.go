// services/order_transitions.go
package services

import (
	"errors"
	"gorm.io/gorm"
)

// ----- Owner actions -----
func (s *OrderService) OwnerAccept(ownerID, orderID uint) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		o, err := s.Repo.GetOrder(orderID); if err != nil { return err }
		ok, err := s.RestRepo.IsOwnedBy(o.RestaurantID, ownerID); if err != nil { return err }
		if !ok { return errors.New("forbidden") }

		affected, err := s.Repo.UpdateStatusGuard(tx, o.ID, s.Status.Pending, s.Status.Preparing)
		if err != nil { return err }
		if affected == 0 { return errors.New("invalid_or_conflict") }
		return nil
	})
}
func (s *OrderService) OwnerHandoffToRider(ownerID, orderID uint) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		o, err := s.Repo.GetOrder(orderID); if err != nil { return err }
		ok, err := s.RestRepo.IsOwnedBy(o.RestaurantID, ownerID); if err != nil { return err }
		if !ok { return errors.New("forbidden") }

		affected, err := s.Repo.UpdateStatusGuard(tx, o.ID, s.Status.Preparing, s.Status.Delivering)
		if err != nil { return err }
		if affected == 0 { return errors.New("invalid_or_conflict") }
		return nil
	})
}
func (s *OrderService) OwnerComplete(ownerID, orderID uint) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		o, err := s.Repo.GetOrder(orderID); if err != nil { return err }
		ok, err := s.RestRepo.IsOwnedBy(o.RestaurantID, ownerID); if err != nil { return err }
		if !ok { return errors.New("forbidden") }

		affected, err := s.Repo.UpdateStatusGuard(tx, o.ID, s.Status.Delivering, s.Status.Completed)
		if err != nil { return err }
		if affected == 0 { return errors.New("invalid_or_conflict") }
		return nil
	})
}
func (s *OrderService) OwnerCancel(ownerID, orderID uint) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		o, err := s.Repo.GetOrder(orderID); if err != nil { return err }
		ok, err := s.RestRepo.IsOwnedBy(o.RestaurantID, ownerID); if err != nil { return err }
		if !ok { return errors.New("forbidden") }

		affected, err := s.Repo.UpdateStatusGuard(tx, o.ID, s.Status.Pending, s.Status.Cancelled)
		if err != nil { return err }
		if affected == 0 { return errors.New("invalid_or_conflict") }
		return nil
	})
}

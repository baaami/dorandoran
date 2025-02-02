package repository

import (
	"errors"
	"log"

	"gorm.io/gorm"
)

type FilterRepository struct {
	db *gorm.DB
}

func NewFilterRepository(db *gorm.DB) *FilterRepository {
	return &FilterRepository{db: db}
}

// 매칭 필터 삽입 또는 업데이트
func (r *FilterRepository) UpsertMatchFilter(filter MatchFilter) error {
	if err := r.db.Save(&filter).Error; err != nil {
		log.Printf("❌ Failed to upsert match filter for user ID %d: %v", filter.UserID, err)
		return err
	}
	return nil
}

// 매칭 필터 조회
func (r *FilterRepository) GetMatchFilterByUserID(userID int) (*MatchFilter, error) {
	var filter MatchFilter
	err := r.db.First(&filter, "user_id = ?", userID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		log.Printf("❌ Failed to get match filter for user ID %d: %v", userID, err)
		return nil, err
	}
	return &filter, nil
}

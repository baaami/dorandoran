package repository

import (
	"errors"
	"log"
	"solo/pkg/models"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// 데이터베이스 초기화
func (r *UserRepository) InitDB() error {
	err := r.db.AutoMigrate(&models.User{}, &models.MatchFilter{})
	if err != nil {
		log.Printf("❌ Failed to migrate tables: %v", err)
		return err
	}
	log.Println("✅ Tables users and matchfilters migrated or already exist.")
	return nil
}

// 유저 생성
func (r *UserRepository) InsertUser(user models.User) (int, error) {
	if err := r.db.Create(&user).Error; err != nil {
		log.Printf("❌ Failed to insert user: %v", err)
		return 0, err
	}
	return user.ID, nil
}

// 유저 조회 (ID)
func (r *UserRepository) GetUserByID(id int) (*models.User, error) {
	var user models.User
	err := r.db.First(&user, id).Error
	if err != nil {
		log.Printf("❌ Failed to get user by ID %d: %v", id, err)
		return nil, err
	}
	return &user, nil
}

// 유저 리스트 조회
func (r *UserRepository) GetUserList() ([]models.User, error) {
	var users []models.User
	err := r.db.Find(&users).Error
	if err != nil {
		log.Printf("❌ Failed to get user list: %v", err)
		return nil, err
	}
	return users, nil
}

// 유저 조회 (SNS)
func (r *UserRepository) GetUserBySNS(snsType int, snsID string) (*models.User, error) {
	var user models.User
	err := r.db.Where("sns_type = ? AND sns_id = ?", snsType, snsID).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		log.Printf("❌ Failed to get user by SNS type %d and SNS ID %s: %v", snsType, snsID, err)
		return nil, err
	}
	return &user, nil
}

// 유저 알람 허용 여부 조회
func (r *UserRepository) GetUserAlert(id int) (bool, error) {
	var user models.User
	err := r.db.Where("id = ?", id).First(&user).Error
	if err != nil {
		log.Printf("❌ Failed to get user alert by ID %d: %v", id, err)
		return false, err
	}
	return user.Alert, nil
}

// 유저 업데이트
func (r *UserRepository) UpdateUser(user models.User) error {
	if err := r.db.Model(&models.User{ID: user.ID}).Select("*").Updates(user).Error; err != nil {
		log.Printf("❌ Failed to update user ID %d: %v", user.ID, err)
		return err
	}
	return nil
}

// 유저 상태 업데이트
func (r *UserRepository) UpdateUserGameInfo(userID int, newStatus int, gameRoomID string) error {
	// 한 번의 쿼리로 game_status와 game_room_id를 함께 업데이트
	result := r.db.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"game_status":  newStatus,
		"game_room_id": gameRoomID,
	})

	if result.Error != nil {
		log.Printf("❌ Failed to update game info for user ID %d: %v", userID, result.Error)
		return result.Error
	}

	if result.RowsAffected == 0 {
		log.Printf("⚠️ No user found with ID %d to update game info", userID)
		return errors.New("user not found")
	}

	return nil
}

// 유저 삭제
func (r *UserRepository) DeleteUser(id int) error {
	if err := r.db.Delete(&models.User{}, id).Error; err != nil {
		log.Printf("❌ Failed to delete user ID %d: %v", id, err)
		return err
	}
	return nil
}

package data

import (
	"errors"
	"log"

	"gorm.io/gorm"
)

type Address struct {
	City     string `gorm:"size:100" json:"city"`
	District string `gorm:"size:100" json:"district"`
	Street   string `gorm:"size:100" json:"street"`
}

type User struct {
	ID         int     `gorm:"primaryKey;autoIncrement" json:"id"`
	SnsType    int     `gorm:"index" json:"sns_type"`
	SnsID      string  `gorm:"index" json:"sns_id"`
	Name       string  `gorm:"size:100" json:"name"`
	Gender     int     `json:"gender"`
	Birth      string  `gorm:"size:20" json:"birth"`
	Address    Address `gorm:"embedded;embeddedPrefix:address_" json:"address"`
	GameStatus int     `gorm:"default:0" json:"game_status"`
	GameRoomID string  `gorm:"size:100" json:"game_room_id"`
	GamePoint  int     `json:"game_point"`
}
type MatchFilter struct {
	UserID          int  `gorm:"primaryKey" json:"user_id"`
	CoupleCount     int  `json:"couple_count"`
	AddressRangeUse bool `json:"address_range_use"`
	AgeGroupUse     bool `json:"age_group_use"`
}

// GORM 클라이언트 설정
type UserService struct {
	DB *gorm.DB
}

// MySQL 데이터베이스 및 테이블 초기화 함수
func (s *UserService) InitDB() error {
	// 데이터베이스 자동 마이그레이션 (테이블 생성)
	err := s.DB.AutoMigrate(&User{}, &MatchFilter{})
	if err != nil {
		log.Printf("Failed to migrate tables: %v", err)
		return err
	}
	log.Println("Tables users and matchfilters migrated or already exist.")
	return nil
}

// 유저 생성 (삽입)
func (s *UserService) InsertUser(user User) (int, error) {
	if err := s.DB.Create(&user).Error; err != nil {
		log.Printf("Failed to insert user: %v", err)
		return 0, err
	}
	return user.ID, nil
}

// 유저 조회
func (s *UserService) GetUserByID(id int) (*User, error) {
	var user User
	err := s.DB.First(&user, id).Error
	if err != nil {
		log.Printf("Failed to get user by ID %d: %v", id, err)
		return nil, err
	}
	return &user, nil
}

// 유저 리스트 조회
func (s *UserService) GetUserList() (*[]User, error) {
	var users []User
	err := s.DB.Find(&users).Error
	if err != nil {
		log.Printf("Failed to get user list: %v", err)
		return nil, err
	}
	return &users, nil
}

// 유저 조회 (sns_type과 sns_id를 기반으로 조회)
func (s *UserService) GetUserBySNS(snsType int, snsID string) (*User, error) {
	var user User
	err := s.DB.Where("sns_type = ? AND sns_id = ?", snsType, snsID).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		log.Printf("Failed to get user by SNS type %d and SNS ID %s: %v", snsType, snsID, err)
		return nil, err
	}
	return &user, nil
}

// 유저 업데이트
func (s *UserService) UpdateUser(user User) error {
	if err := s.DB.Model(&User{ID: user.ID}).Updates(user).Error; err != nil {
		log.Printf("Failed to update user ID %d: %v", user.ID, err)
		return err
	}
	return nil
}

// 유저 상태 업데이트
func (s *UserService) UpdateUserStatus(userID int, newStatus int) error {
	result := s.DB.Model(&User{}).Where("id = ?", userID).Update("game_status", newStatus)
	if result.Error != nil {
		log.Printf("Failed to update game_status for user ID %d: %v", userID, result.Error)
		return result.Error
	}
	if result.RowsAffected == 0 {
		log.Printf("No user found with ID %d to update game_status", userID)
		return errors.New("user not found")
	}
	return nil
}

// 유저 현재 게임방 업데이트
func (s *UserService) UpdateUserGameRoomID(userID int, gameRoomID string) error {
	result := s.DB.Model(&User{}).Where("id = ?", userID).Update("game_room_id", gameRoomID)
	if result.Error != nil {
		log.Printf("Failed to update game_room_id for user ID %d: %v", userID, result.Error)
		return result.Error
	}
	if result.RowsAffected == 0 {
		log.Printf("No user found with ID %d to update game_room_id", userID)
		return errors.New("user not found")
	}
	return nil
}

// 유저 삭제
func (s *UserService) DeleteUser(id int) error {
	if err := s.DB.Delete(&User{}, id).Error; err != nil {
		log.Printf("Failed to delete user ID %d: %v", id, err)
		return err
	}
	return nil
}

// 매칭 필터 삽입 또는 업데이트
func (s *UserService) UpsertMatchFilter(filter MatchFilter) (MatchFilter, error) {
	if err := s.DB.Save(&filter).Error; err != nil {
		log.Printf("Failed to upsert match filter for user ID %d: %v", filter.UserID, err)
		return MatchFilter{}, err
	}
	return filter, nil
}

// 매칭 필터 조회
func (s *UserService) GetMatchFilterByUserID(userID int) (*MatchFilter, error) {
	var filter MatchFilter
	err := s.DB.First(&filter, "user_id = ?", userID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		log.Printf("Failed to get match filter for user ID %d: %v", userID, err)
		return nil, err
	}
	return &filter, nil
}

func (s *UserService) IncreaseGamePoint(userID int, points int) error {
	if points <= 0 {
		return errors.New("points to increase must be greater than zero")
	}
	if err := s.DB.Model(&User{}).Where("id = ?", userID).Update("game_point", gorm.Expr("game_point + ?", points)).Error; err != nil {
		log.Printf("Failed to increase game points for user ID %d: %v", userID, err)
		return err
	}
	return nil
}

func (s *UserService) DecreaseGamePoint(userID int, points int) error {
	if points <= 0 {
		return errors.New("points to decrease must be greater than zero")
	}
	if err := s.DB.Model(&User{}).Where("id = ? AND game_point >= ?", userID, points).Update("game_point", gorm.Expr("game_point - ?", points)).Error; err != nil {
		log.Printf("Failed to decrease game points for user ID %d: %v", userID, err)
		return err
	}
	return nil
}

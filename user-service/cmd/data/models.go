package data

import (
	"log"

	"gorm.io/gorm"
)

type User struct {
	ID       int    `gorm:"primaryKey;autoIncrement" json:"id"`
	SnsType  int    `gorm:"index" json:"sns_type"`
	SnsID    int64  `gorm:"index" json:"sns_id"`
	Name     string `gorm:"size:100" json:"name"`
	Nickname string `gorm:"size:100" json:"nickname"`
	Gender   int    `json:"gender"`
	Age      int    `json:"age"`
	Email    string `gorm:"size:100" json:"email"`
}

// GORM 클라이언트 설정
type UserService struct {
	DB *gorm.DB
}

// MySQL 데이터베이스 및 테이블 초기화 함수
func (s *UserService) InitDB() error {
	// 데이터베이스 자동 마이그레이션 (테이블 생성)
	err := s.DB.AutoMigrate(&User{})
	if err != nil {
		return err
	}
	log.Println("Table `users` migrated or already exists.")
	return nil
}

// 유저 생성 (삽입)
func (s *UserService) InsertUser(name, nickname string, snsID int64, gender, age, snsType int, email string) (int64, error) {
	user := User{
		Name:     name,
		Nickname: nickname,
		SnsID:    snsID,
		Gender:   gender,
		Age:      age,
		SnsType:  snsType,
		Email:    email,
	}
	if err := s.DB.Create(&user).Error; err != nil {
		return 0, err
	}

	log.Printf("[DB] isnert user: %v", user)
	return int64(user.ID), nil
}

// 유저 조회
func (s *UserService) GetUserByID(id int) (*User, error) {
	var user User
	err := s.DB.First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// 유저 조회 (sns_type과 sns_id를 기반으로 조회)
func (s *UserService) GetUserBySNS(snsType int, snsID int64) (*User, error) {
	var user User
	err := s.DB.Where("sns_type = ? AND sns_id = ?", snsType, snsID).First(&user).Error
	if err != nil {
		// record not found인 경우 user와 error 모두 nil 반환
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		// 다른 에러가 있는 경우 에러 반환
		return nil, err
	}
	return &user, nil
}

// 유저 업데이트
func (s *UserService) UpdateUser(id int, name, nickname string, gender, age int) error {
	user := User{
		ID: id,
	}
	updateFields := map[string]interface{}{
		"name":     name,
		"nickname": nickname,
		"gender":   gender,
		"age":      age,
	}
	if err := s.DB.Model(&user).Updates(updateFields).Error; err != nil {
		return err
	}
	return nil
}

// 유저 삭제
func (s *UserService) DeleteUser(id int) error {
	if err := s.DB.Delete(&User{}, id).Error; err != nil {
		return err
	}
	return nil
}

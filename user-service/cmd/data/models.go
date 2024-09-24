package data

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

type User struct {
	ID       int    `json:"id"`
	SnsType  int    `json:"sns_type"`
	SnsID    string `json:"sns_id"`
	Name     string `json:"name"`
	Nickname string `json:"nickname"`
	Gender   int    `json:"gender"`
	Age      int    `json:"age"`
	Email    string `json:"email"`
}

// MySQL 클라이언트 설정
type UserService struct {
	DB *sql.DB
}

// MySQL 데이터베이스 연결 함수
func (s *UserService) InitDB() error {
	// 데이터베이스 생성 쿼리 실행
	_, err := s.DB.Exec("CREATE DATABASE IF NOT EXISTS users")
	if err != nil {
		return fmt.Errorf("failed to create database: %v", err)
	}
	log.Println("Database `users` created or already exists.")

	// `users` 데이터베이스 사용
	_, err = s.DB.Exec("USE users")
	if err != nil {
		return fmt.Errorf("failed to switch to database `users`: %v", err)
	}

	// `users` 테이블 생성 쿼리
	tableCreationQuery := `
	CREATE TABLE IF NOT EXISTS users (
		id INT AUTO_INCREMENT PRIMARY KEY,
		sns_type INT,
		sns_id VARCHAR(255),
		name VARCHAR(100),
		nickname VARCHAR(100),
		gender INT,
		age INT,
		email VARCHAR(100)
	);`

	// 테이블 생성 쿼리 실행
	_, err = s.DB.Exec(tableCreationQuery)
	if err != nil {
		return fmt.Errorf("failed to create table: %v", err)
	}
	log.Println("Table `users` created or already exists.")

	return nil
}

// 유저 생성 (삽입)
func (s *UserService) InsertUser(name, nickname, snsID string, gender, age, snsType int, email string) (int64, error) {
	query := "INSERT INTO users (name, nickname, sns_id, gender, age, sns_type, email) VALUES (?, ?, ?, ?, ?, ?, ?)"
	result, err := s.DB.Exec(query, name, nickname, snsID, gender, age, snsType, email)
	if err != nil {
		return 0, fmt.Errorf("failed to insert user: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to retrieve last insert id: %v", err)
	}
	return id, nil
}

// 유저 조회
func (s *UserService) GetUserByID(id int) (*User, error) {
	query := "SELECT id, name, nickname, gender, age, email FROM users WHERE id = ?"
	row := s.DB.QueryRow(query, id)

	var user User
	err := row.Scan(&user.ID, &user.Name, &user.Nickname, &user.Gender, &user.Age, &user.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // 유저가 없을 경우 nil 반환
		}
		return nil, fmt.Errorf("failed to retrieve user: %v", err)
	}
	return &user, nil
}

// 유저 조회 (sns_type과 sns_id를 기반으로 조회)
func (s *UserService) GetUserBySNS(snsType int, snsID string) (*User, error) {
	query := "SELECT id, sns_type, sns_id, name, nickname, gender, age, email FROM users WHERE sns_type = ? AND sns_id = ?"
	row := s.DB.QueryRow(query, snsType, snsID)

	var user User
	err := row.Scan(&user.ID, &user.SnsType, &user.SnsID, &user.Name, &user.Nickname, &user.Gender, &user.Age, &user.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // 유저가 없을 경우 nil 반환
		}
		return nil, fmt.Errorf("failed to retrieve user by sns_type and sns_id: %v", err)
	}
	return &user, nil
}

// 유저 업데이트
func (s *UserService) UpdateUser(id int, name, nickname string, gender, age int) error {
	query := "UPDATE users SET name = ?, nickname = ?, gender = ?, age = ? WHERE id = ?"
	_, err := s.DB.Exec(query, name, nickname, gender, age, id)
	if err != nil {
		return fmt.Errorf("failed to update user: %v", err)
	}
	return nil
}

// 유저 삭제
func (s *UserService) DeleteUser(id int) error {
	query := "DELETE FROM users WHERE id = ?"
	_, err := s.DB.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %v", err)
	}
	return nil
}

package db

import (
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ConnectMySQL: MySQL 연결을 설정하고 반환
func ConnectMySQL() (*gorm.DB, error) {
	// 환경 변수에서 DSN 가져오기
	dsn := GetMySQLDSN()
	if dsn == "" {
		log.Fatal("❌ MySQL DSN이 설정되지 않았습니다.")
	}

	// MySQL 연결
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Printf("❌ MySQL 연결 실패: %v", err)
		return nil, err
	}

	log.Println("✅ MySQL 연결 성공!")
	return db, nil
}

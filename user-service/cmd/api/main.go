package main

import (
	"fmt"
	"log"
	"net/http"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/baaami/dorandoran/user/cmd/data"
)

const webPort = 80

type Config struct {
	Models *data.UserService
}

func main() {
	mysqlClient, err := connectToMySQL()
	if err != nil {
		log.Panic(err)
	}

	// Config 구조체 초기화
	app := Config{
		Models: &data.UserService{DB: mysqlClient},
	}

	// DB 초기화 (데이터베이스 및 테이블 생성)
	err = app.Models.InitDB()
	if err != nil {
		log.Panic(err)
	}

	// HTTP 서버 설정
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", webPort),
		Handler: app.routes(),
	}

	// 서버 시작
	log.Printf("Starting User Service on port %d", webPort)
	err = srv.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}

// MySQL에 연결하는 함수
func connectToMySQL() (*gorm.DB, error) {
	dsn := "root:sample@tcp(mysql:3306)/users?parseTime=true"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info), // 로깅 설정
	})
	if err != nil {
		log.Println("Error connecting to MySQL:", err)
		return nil, err
	}

	log.Println("Connected to MySQL!")

	return db, nil
}

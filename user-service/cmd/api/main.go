package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/baaami/dorandoran/user/cmd/data"
)

const webPort = 80

var db *sql.DB

type Config struct {
	Models *data.UserService
}

func main() {
	mysqlClient, err := connectToMySQL()
	if err != nil {
		log.Panic(err)
	}
	db = mysqlClient

	// MySQL 연결 해제 시 사용되는 컨텍스트 생성
	_, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// MySQL 연결 해제
	defer func() {
		if err = db.Close(); err != nil {
			panic(err)
		}
	}()

	// Config 구조체 초기화
	app := Config{
		Models: &data.UserService{DB: db},
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
func connectToMySQL() (*sql.DB, error) {
	// MySQL 데이터 소스 네임 (DSN) 설정
	dsn := "root:sample@tcp(mysql:3306)/users?parseTime=true"

	// MySQL에 연결
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Println("Error connecting to MySQL:", err)
		return nil, err
	}

	// MySQL 연결 확인
	if err := db.Ping(); err != nil {
		log.Println("Error pinging MySQL:", err)
		return nil, err
	}

	log.Println("Connected to MySQL!")

	return db, nil
}

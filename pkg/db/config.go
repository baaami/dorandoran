package db

import (
	"os"
)

// GetMySQLDSN: 환경 변수에서 DSN 가져오기
func GetMySQLDSN() string {
	// 환경 변수에서 DSN 값 가져오기
	dsn := os.Getenv("MYSQL_DSN")

	// 기본값 설정 (환경 변수가 없을 경우)
	if dsn == "" {
		dsn = "root:sample@tcp(mysql:3306)/users?parseTime=true"
	}

	return dsn
}

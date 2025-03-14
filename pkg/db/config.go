package db

import (
	"fmt"
	"os"
)

// GetMySQLDSN: 환경 변수에서 DSN 가져오기
func GetMySQLDSN() string {
	if dsn := os.Getenv("MYSQL_DSN"); dsn != "" {
		return dsn
	}

	host := os.Getenv("MYSQL_HOST")
	if host == "" {
		host = "doran-mysql"
	}

	return fmt.Sprintf("root:sample@tcp(%s:3306)/users?parseTime=true", host)
}

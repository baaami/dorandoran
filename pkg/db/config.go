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

	user := os.Getenv("MYSQL_ROOT_USER")
	if user == "" {
		user = "root"
	}

	password := os.Getenv("MYSQL_ROOT_PASSWORD")
	if password == "" {
		password = "sample"
	}

	return fmt.Sprintf("%s:%s@tcp(%s:3306)/users?parseTime=true", user, password, host)
}

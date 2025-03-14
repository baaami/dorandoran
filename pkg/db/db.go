package db

import (
	"gorm.io/gorm"
)

type DB interface {
	GetDB() *gorm.DB
}

// MySQLDatabase 구조체
type MySQLDatabase struct {
	DB *gorm.DB
}

func (m *MySQLDatabase) GetDB() *gorm.DB {
	return m.DB
}

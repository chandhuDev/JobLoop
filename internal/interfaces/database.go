package interfaces

import "gorm.io/gorm"

type DatabaseClient interface {
	CreateSchema() error
	GetDB() *gorm.DB
}

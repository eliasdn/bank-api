package db

import "gorm.io/gorm"

type DBInterface interface {
	First(interface{}, ...interface{}) *gorm.DB
	Create(interface{}) *gorm.DB
	Where(interface{}, ...interface{}) *gorm.DB
	Model(interface{}) *gorm.DB
	Update(string, interface{}) *gorm.DB
	Delete(interface{}, ...interface{}) *gorm.DB
	Find(interface{}, ...interface{}) *gorm.DB
	Preload(string, ...interface{}) *gorm.DB
	Begin() *gorm.DB
	Commit() *gorm.DB
	Rollback() *gorm.DB
	Error() error
	RowsAffected() int64
	Order(interface{}) *gorm.DB
	Save(interface{}) *gorm.DB
	GetDB() *gorm.DB
}

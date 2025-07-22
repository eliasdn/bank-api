package db

import "gorm.io/gorm"

// DBAdapter implements DBInterface using gorm.DB
type DBAdapter struct {
	DB *gorm.DB
}

// NewDBAdapter creates a new DBAdapter
func NewDBAdapter(db *gorm.DB) *DBAdapter {
	return &DBAdapter{DB: db}
}

// DBInterface implementation
func (d *DBAdapter) First(out interface{}, where ...interface{}) *gorm.DB {
	return d.DB.First(out, where...)
}

func (d *DBAdapter) Create(value interface{}) *gorm.DB {
	return d.DB.Create(value)
}

func (d *DBAdapter) Where(query interface{}, args ...interface{}) *gorm.DB {
	return d.DB.Where(query, args...)
}

func (d *DBAdapter) Model(value interface{}) *gorm.DB {
	return d.DB.Model(value)
}

func (d *DBAdapter) Update(column string, value interface{}) *gorm.DB {
	return d.DB.Update(column, value)
}

func (d *DBAdapter) Delete(value interface{}, where ...interface{}) *gorm.DB {
	return d.DB.Delete(value, where...)
}

func (d *DBAdapter) Find(dest interface{}, conds ...interface{}) *gorm.DB {
	return d.DB.Find(dest, conds...)
}

func (d *DBAdapter) Preload(column string, conditions ...interface{}) *gorm.DB {
	return d.DB.Preload(column, conditions...)
}

func (d *DBAdapter) Begin() *gorm.DB {
	return d.DB.Begin()
}

func (d *DBAdapter) Commit() *gorm.DB {
	return d.DB.Commit()
}

func (d *DBAdapter) Rollback() *gorm.DB {
	return d.DB.Rollback()
}

func (d *DBAdapter) Error() error {
	return d.DB.Error
}

func (d *DBAdapter) RowsAffected() int64 {
	return d.DB.RowsAffected
}

func (d *DBAdapter) Order(value interface{}) *gorm.DB {
	return d.DB.Order(value)
}

func (d *DBAdapter) Save(value interface{}) *gorm.DB {
	return d.DB.Save(value)
}

func (d *DBAdapter) GetDB() *gorm.DB {
	return d.DB
}

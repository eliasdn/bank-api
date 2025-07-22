package db

import (
	"bank-api/internal/config"
	"bank-api/internal/errors"
	"bank-api/internal/models"
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Database struct {
	DB *gorm.DB
}

// Ensure Database implements DBInterface
var _ DBInterface = (*Database)(nil)

func InitDB(cfg *config.AppConfig) (*Database, error) {
	// Configure GORM with SQLite
	db, err := gorm.Open(sqlite.Open("bank.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrDBConnection.Code, "failed to connect to database", errors.ErrDBConnection.Status)
	}

	// Configure connection pool for SQLite
	sqlDB, err := db.DB()
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrDBConnection.Code, "failed to get sql.DB", errors.ErrDBConnection.Status)
	}

	// Set connection pool settings from configuration
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.Database.ConnMaxIdleTime)

	// Run migrations
	err = RunMigrations("bank.db")
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrDBMigration.Code, "failed to run migrations", errors.ErrDBMigration.Status)
	}

	log.Println("Database initialized successfully")
	return &Database{DB: db}, nil
}

// DBInterface implementation
func (d *Database) First(out interface{}, where ...interface{}) *gorm.DB {
	return d.DB.First(out, where...)
}

func (d *Database) Create(value interface{}) *gorm.DB {
	return d.DB.Create(value)
}

func (d *Database) Where(query interface{}, args ...interface{}) *gorm.DB {
	return d.DB.Where(query, args...)
}

func (d *Database) Model(value interface{}) *gorm.DB {
	return d.DB.Model(value)
}

func (d *Database) Update(column string, value interface{}) *gorm.DB {
	return d.DB.Update(column, value)
}

func (d *Database) Delete(value interface{}, where ...interface{}) *gorm.DB {
	return d.DB.Delete(value, where...)
}

func (d *Database) Find(out interface{}, where ...interface{}) *gorm.DB {
	return d.DB.Find(out, where...)
}

func (d *Database) Preload(column string, conditions ...interface{}) *gorm.DB {
	return d.DB.Preload(column, conditions...)
}

func (d *Database) Begin() *gorm.DB {
	return d.DB.Begin()
}

func (d *Database) Commit() *gorm.DB {
	return d.DB.Commit()
}

func (d *Database) Rollback() *gorm.DB {
	return d.DB.Rollback()
}

func (d *Database) Error() error {
	return d.DB.Error
}

func (d *Database) RowsAffected() int64 {
	return d.DB.RowsAffected
}

func (d *Database) Order(value interface{}) *gorm.DB {
	return d.DB.Order(value)
}

func (d *Database) Save(value interface{}) *gorm.DB {
	return d.DB.Save(value)
}

// GetDB returns the underlying gorm.DB instance
func (d *Database) GetDB() *gorm.DB {
	return d.DB
}

// Close closes the database connection
func (d *Database) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Health check
func (d *Database) Health() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

// UserRepository methods
func (d *Database) CreateUser(user *models.User) error {
	return d.DB.Create(user).Error
}

func (d *Database) GetUserByID(id uint) (*models.User, error) {
	var user models.User
	err := d.DB.First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (d *Database) GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	err := d.DB.Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (d *Database) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	err := d.DB.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (d *Database) UpdateUser(user *models.User) error {
	return d.DB.Save(user).Error
}

func (d *Database) DeleteUser(id uint) error {
	return d.DB.Delete(&models.User{}, id).Error
}

// AccountRepository methods
func (d *Database) CreateAccount(account *models.Account) error {
	return d.DB.Create(account).Error
}

func (d *Database) GetAccountByID(id uint) (*models.Account, error) {
	var account models.Account
	err := d.DB.First(&account, id).Error
	if err != nil {
		return nil, err
	}
	return &account, nil
}

func (d *Database) GetAccountsByUserID(userID uint) ([]models.Account, error) {
	var accounts []models.Account
	err := d.DB.Where("user_id = ?", userID).Find(&accounts).Error
	if err != nil {
		return nil, err
	}
	return accounts, nil
}

func (d *Database) UpdateAccount(account *models.Account) error {
	return d.DB.Save(account).Error
}

func (d *Database) DeleteAccount(id uint) error {
	return d.DB.Delete(&models.Account{}, id).Error
}

func (d *Database) UpdateAccountBalance(accountID uint, amount float64, tx *gorm.DB) error {
	db := d.DB
	if tx != nil {
		db = tx
	}

	return db.Model(&models.Account{}).
		Where("id = ?", accountID).
		Update("balance", gorm.Expr("balance + ?", amount)).Error
}

// TransactionRepository methods
func (d *Database) CreateTransaction(transaction *models.Transaction) error {
	return d.DB.Create(transaction).Error
}

func (d *Database) GetTransactionByID(id uint) (*models.Transaction, error) {
	var transaction models.Transaction
	err := d.DB.First(&transaction, id).Error
	if err != nil {
		return nil, err
	}
	return &transaction, nil
}

func (d *Database) GetTransactionsByAccountID(accountID uint) ([]models.Transaction, error) {
	var transactions []models.Transaction
	err := d.DB.Where("account_id = ?", accountID).
		Order("created_at DESC").
		Find(&transactions).Error
	if err != nil {
		return nil, err
	}
	return transactions, nil
}

func (d *Database) GetTransactionsByUserID(userID uint) ([]models.Transaction, error) {
	var transactions []models.Transaction
	err := d.DB.Joins("JOIN accounts ON transactions.account_id = accounts.id").
		Where("accounts.user_id = ?", userID).
		Order("transactions.created_at DESC").
		Find(&transactions).Error
	if err != nil {
		return nil, err
	}
	return transactions, nil
}

func (d *Database) UpdateTransaction(transaction *models.Transaction) error {
	return d.DB.Save(transaction).Error
}

func (d *Database) DeleteTransaction(id uint) error {
	return d.DB.Delete(&models.Transaction{}, id).Error
}

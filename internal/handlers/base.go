package handlers

import (
	"bank-api/internal/config"
	"bank-api/internal/db"
	"bank-api/internal/services"
	"golang.org/x/crypto/bcrypt"
	"sync"
)

type Handler struct {
	DB           db.DBInterface
	Config       *config.AppConfig
	AuditService *services.AuditService
	dummyHash    string
	TxCountCache sync.Map
}

func NewHandler(db db.DBInterface, cfg *config.AppConfig) *Handler {
	// Generate dummy hash for timing attack prevention
	// We use the same cost as regular passwords
	dummyHash, _ := bcrypt.GenerateFromPassword([]byte("dummy_password_for_timing_prevention"), cfg.Security.BcryptCost)

	return &Handler{
		DB:           db,
		Config:       cfg,
		AuditService: services.NewAuditService(db.GetDB()),
		dummyHash:    string(dummyHash),
		TxCountCache: sync.Map{},
	}
}

package handlers

import (
	"bank-api/internal/config"
	"bank-api/internal/db"
	"bank-api/internal/services"
)

type Handler struct {
	DB           db.DBInterface
	Config       *config.AppConfig
	AuditService *services.AuditService
}

func NewHandler(db db.DBInterface, cfg *config.AppConfig) *Handler {
	return &Handler{
		DB:           db,
		Config:       cfg,
		AuditService: services.NewAuditService(db.GetDB()),
	}
}

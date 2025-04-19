package handler

import (
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/location"
)

// LocationHandler handles location related operations
type LocationHandler struct {
	locationUC location.LocationUC
	cfg        *models.Config
}

// NewLocationHandler creates a new location handler
func NewLocationHandler(
	locationUC location.LocationUC,
	cfg *models.Config,
) *LocationHandler {
	return &LocationHandler{
		locationUC: locationUC,
		cfg:        cfg}
}

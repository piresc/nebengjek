package handler

import (
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/rides"
)

// MatchHandler handles requests for the match service
type ridesHandler struct {
	cfg     *models.Config
	ridesUC rides.RideUC
}

// NewMatchHandler creates a new match handler
func NewRideHandler(
	cfg *models.Config,
	ridesUC rides.RideUC,
) *ridesHandler {
	return &ridesHandler{
		cfg:     cfg,
		ridesUC: ridesUC,
	}
}

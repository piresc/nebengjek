package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/piresc/nebengjek/match-service/usecase"
)

type MatchHandler struct {
	matchUseCase *usecase.MatchUseCase
}

func NewMatchHandler(matchUseCase *usecase.MatchUseCase) *MatchHandler {
	return &MatchHandler{
		matchUseCase: matchUseCase,
	}
}

type MatchRequest struct {
	UserID               string  `json:"user_id" binding:"required"`
	PickupLatitude       float64 `json:"pickup_latitude" binding:"required"`
	PickupLongitude      float64 `json:"pickup_longitude" binding:"required"`
	DestinationLatitude  float64 `json:"destination_latitude" binding:"required"`
	DestinationLongitude float64 `json:"destination_longitude" binding:"required"`
}

type LocationUpdateRequest struct {
	DriverID  string  `json:"driver_id" binding:"required"`
	Latitude  float64 `json:"latitude" binding:"required"`
	Longitude float64 `json:"longitude" binding:"required"`
	Status    string  `json:"status" binding:"required"`
}

type NearbyDriversRequest struct {
	Latitude  float64 `json:"latitude" binding:"required"`
	Longitude float64 `json:"longitude" binding:"required"`
	RadiusKm  float64 `json:"radius_km" binding:"required"`
}

func (h *MatchHandler) SetupRoutes(router *gin.Engine) {
	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Match routes
	match := router.Group("/match")
	{
		match.POST("/request", h.HandleMatchRequest)
		match.POST("/location/update", h.HandleLocationUpdate)
		match.GET("/nearby-drivers", h.HandleNearbyDrivers)
	}
}

func (h *MatchHandler) HandleMatchRequest(c *gin.Context) {
	var req MatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	match, err := h.matchUseCase.RequestMatch(
		c.Request.Context(),
		req.UserID,
		req.PickupLatitude,
		req.PickupLongitude,
		req.DestinationLatitude,
		req.DestinationLongitude,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, match)
}

func (h *MatchHandler) HandleLocationUpdate(c *gin.Context) {
	var req LocationUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.matchUseCase.UpdateDriverLocation(
		c.Request.Context(),
		req.DriverID,
		req.Latitude,
		req.Longitude,
		req.Status,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Location updated successfully"})
}

func (h *MatchHandler) HandleNearbyDrivers(c *gin.Context) {
	var req NearbyDriversRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	drivers, err := h.matchUseCase.GetNearbyDrivers(
		c.Request.Context(),
		req.Latitude,
		req.Longitude,
		req.RadiusKm,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"drivers": drivers})
}

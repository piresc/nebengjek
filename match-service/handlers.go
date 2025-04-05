package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/piresc/nebengjek/match-service/proto"
)

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

func setupRESTRoutes(router *gin.Engine, server *MatchServer) {
	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Match routes
	match := router.Group("/match")
	{
		match.POST("/request", handleMatchRequest(server))
		match.POST("/location/update", handleLocationUpdate(server))
		match.GET("/nearby-drivers", handleNearbyDrivers(server))
	}
}

func handleMatchRequest(server *MatchServer) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req MatchRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		protoReq := &proto.MatchRequest{
			UserId:               req.UserID,
			PickupLatitude:       req.PickupLatitude,
			PickupLongitude:      req.PickupLongitude,
			DestinationLatitude:  req.DestinationLatitude,
			DestinationLongitude: req.DestinationLongitude,
		}

		resp, err := server.RequestMatch(c.Request.Context(), protoReq)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, resp)
	}
}

func handleLocationUpdate(server *MatchServer) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req LocationUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		protoReq := &proto.LocationUpdate{
			DriverId:  req.DriverID,
			Latitude:  req.Latitude,
			Longitude: req.Longitude,
			Status:    req.Status,
		}

		resp, err := server.UpdateDriverLocation(c.Request.Context(), protoReq)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, resp)
	}
}

func handleNearbyDrivers(server *MatchServer) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req NearbyDriversRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		protoReq := &proto.NearbyDriversRequest{
			Latitude:  req.Latitude,
			Longitude: req.Longitude,
			RadiusKm:  req.RadiusKm,
		}

		resp, err := server.GetNearbyDrivers(c.Request.Context(), protoReq)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, resp)
	}
}

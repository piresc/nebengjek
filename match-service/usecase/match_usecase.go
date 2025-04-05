package usecase

import (
	"context"
	"time"

	"github.com/piresc/nebengjek/match-service/domain/entity"
	"github.com/piresc/nebengjek/match-service/domain/errors"
	"github.com/piresc/nebengjek/match-service/domain/repository"
)

type MatchUseCase struct {
	driverRepo repository.DriverRepository
	matchRepo  repository.MatchRepository
}

func NewMatchUseCase(driverRepo repository.DriverRepository, matchRepo repository.MatchRepository) *MatchUseCase {
	return &MatchUseCase{
		driverRepo: driverRepo,
		matchRepo:  matchRepo,
	}
}

func (u *MatchUseCase) RequestMatch(ctx context.Context, userID string, pickupLat, pickupLon, destLat, destLon float64) (*entity.Match, error) {
	// Validate input
	if userID == "" {
		return nil, errors.ErrInvalidUserID
	}

	if !isValidLocation(pickupLat, pickupLon) || !isValidLocation(destLat, destLon) {
		return nil, errors.ErrInvalidLocation
	}

	// Find nearby drivers
	drivers, err := u.driverRepo.FindNearbyDrivers(ctx, pickupLat, pickupLon, 5.0) // 5km radius
	if err != nil {
		return nil, err
	}

	if len(drivers) == 0 {
		return nil, errors.ErrNoDriversAvailable
	}

	// Select the closest driver (first one, as they're sorted by distance)
	selectedDriver := drivers[0]

	// Create match
	match := &entity.Match{
		ID:              generateMatchID(),
		UserID:          userID,
		DriverID:        selectedDriver.ID,
		Status:          "pending",
		EtaMinutes:      calculateETA(selectedDriver.Distance),
		PickupLatitude:  pickupLat,
		PickupLongitude: pickupLon,
		DestLatitude:    destLat,
		DestLongitude:   destLon,
	}

	if err := u.matchRepo.Create(ctx, match); err != nil {
		return nil, err
	}

	return match, nil
}

func (u *MatchUseCase) UpdateDriverLocation(ctx context.Context, driverID string, lat, lon float64, status string) error {
	if driverID == "" {
		return errors.ErrInvalidDriverID
	}

	if !isValidLocation(lat, lon) {
		return errors.ErrInvalidLocation
	}

	return u.driverRepo.UpdateLocation(ctx, driverID, lat, lon, status)
}

func (u *MatchUseCase) GetNearbyDrivers(ctx context.Context, lat, lon, radiusKm float64) ([]*entity.Driver, error) {
	if !isValidLocation(lat, lon) {
		return nil, errors.ErrInvalidLocation
	}

	if radiusKm <= 0 {
		radiusKm = 5.0 // Default radius
	}

	return u.driverRepo.FindNearbyDrivers(ctx, lat, lon, radiusKm)
}

// Helper function to validate coordinates
func isValidLocation(lat, lon float64) bool {
	return lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180
}

func generateMatchID() string {
	return time.Now().Format("20060102150405") + "-" + generateRandomString(6)
}

func generateRandomString(n int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[time.Now().UnixNano()%int64(len(letterBytes))]
	}
	return string(b)
}

func calculateETA(distanceKm float64) float64 {
	// Simplified ETA calculation (assuming average speed of 30 km/h)
	return (distanceKm / 30.0) * 60.0
}

package repository

import (
	"context"

	"github.com/piresc/nebengjek/match-service/domain/entity"
)

type DriverRepository interface {
	UpdateLocation(ctx context.Context, driverID string, lat, lon float64, status string) error
	FindNearbyDrivers(ctx context.Context, lat, lon, radiusKm float64) ([]*entity.Driver, error)
}

type MatchRepository interface {
	Create(ctx context.Context, match *entity.Match) error
	UpdateStatus(ctx context.Context, matchID string, status string) error
	Get(ctx context.Context, matchID string) (*entity.Match, error)
}

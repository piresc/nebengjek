package repository

import (
	"context"
)

type LocationRepo interface {
	CalculateDistance(ctx context.Context, startLocation, endLocation string) (float64, error)
}

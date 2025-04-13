package usecase

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// HandleBeaconEvent processes beacon events from NATS
func (uc *MatchUC) HandleBeaconEvent(event models.BeaconEvent) error {

	ctx := context.Background()

	// Process based on user role and status
	if event.IsActive {
		// Add user to available pool
		location := &models.Location{
			Latitude:  event.Location.Latitude,
			Longitude: event.Location.Longitude,
			Timestamp: time.Now(),
		}

		var err error
		if event.Role == "driver" {
			// Add driver to available pool and look for nearby passengers
			err = uc.repo.AddAvailableDriver(ctx, event.MSISDN, location)
			if err != nil {
				log.Printf("Failed to add available driver: %v", err)
				return err
			}

			// Find nearby passengers to match with
			matches, err := uc.repo.FindNearbyPassengers(ctx, location, 1.0) // 1km radius
			if err != nil {
				log.Printf("Failed to find nearby passengers: %v", err)
				return err
			}

			// Create match proposals for each nearby passenger
			for _, passengerID := range matches {
				match := &models.Trip{
					PassengerMSISDN: passengerID,
					DriverMSISDN:    event.MSISDN,
					Status:          models.TripStatusProposed,
					RequestedAt:     time.Now(),
					PickupLocation: models.Location{
						Latitude:  location.Latitude,
						Longitude: location.Longitude,
						Timestamp: time.Now(),
					},
				}

				if err := uc.CreateMatch(ctx, match); err != nil {
					log.Printf("Failed to create match for driver %s and passenger %s: %v",
						event.MSISDN, passengerID, err)
					continue
				}
			}
		} else {
			// Add passenger to available pool and look for nearby drivers
			err = uc.repo.AddAvailablePassenger(ctx, event.MSISDN, location)
			if err != nil {
				log.Printf("Failed to add available passenger: %v", err)
				return err
			}

			// Find nearby drivers to match with
			matches, err := uc.repo.FindNearbyDrivers(ctx, location, 1.0) // 1km radius
			if err != nil {
				log.Printf("Failed to find nearby drivers: %v", err)
				return err
			}

			// Create match proposals for each nearby driver
			for _, driverID := range matches {
				match := &models.Trip{
					PassengerMSISDN: event.MSISDN,
					DriverMSISDN:    driverID,
					Status:          models.TripStatusProposed,
					RequestedAt:     time.Now(),
					PickupLocation: models.Location{
						Latitude:  location.Latitude,
						Longitude: location.Longitude,
						Timestamp: time.Now(),
					},
				}

				if err := uc.CreateMatch(ctx, match); err != nil {
					log.Printf("Failed to create match for passenger %s and driver %s: %v",
						event.MSISDN, driverID, err)
					continue
				}
			}
		}
	} else {
		// Remove user from available pool
		if event.Role == "driver" {
			err := uc.repo.RemoveAvailableDriver(ctx, event.MSISDN)
			if err != nil {
				log.Printf("Failed to remove available driver: %v", err)
				return err
			}
		} else {
			err := uc.repo.RemoveAvailablePassenger(ctx, event.MSISDN)
			if err != nil {
				log.Printf("Failed to remove available passenger: %v", err)
				return err
			}
		}
	}

	return nil
}

func (uc *MatchUC) CreateMatch(ctx context.Context, match *models.Trip) error {
	if err := uc.repo.CreateMatch(ctx, match); err != nil {
		return fmt.Errorf("failed to create match: %w", err)
	}
	return nil
}

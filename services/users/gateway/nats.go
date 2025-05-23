package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"log" // Imported log package

	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
	// Assuming UserGW struct and natsClient are defined in init.go or similar in this package
)

// PublishBeaconEvent publishes a beacon event to NATS.
func (g *UserGW) PublishBeaconEvent(ctx context.Context, event *models.BeaconEvent) error {
	subject := constants.SubjectUserBeacon
	log.Printf("Publishing BeaconEvent for UserID %s to subject %s", event.UserID, subject)
	log.Printf("BeaconEvent details: %+v", event)

	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("Error marshalling BeaconEvent for UserID %s: %v", event.UserID, err)
		return fmt.Errorf("failed to marshal BeaconEvent for UserID %s: %w", event.UserID, err)
	}

	if err := g.natsClient.Publish(subject, data); err != nil {
		log.Printf("Error publishing BeaconEvent for UserID %s to NATS subject %s: %v", event.UserID, subject, err)
		return fmt.Errorf("failed to publish BeaconEvent for UserID %s to subject %s: %w", event.UserID, subject, err)
	}

	log.Printf("Successfully published BeaconEvent for UserID %s to subject %s", event.UserID, subject)
	return nil
}

// MatchAccept publishes a match acceptance event from the User service (driver accepting a proposal) to NATS.
// The caller (User service use case) is responsible for ensuring mp.MatchStatus is
// models.MatchStatusPendingCustomerConfirmation if this is the driver's initial acceptance.
func (g *UserGW) MatchAccept(mp *models.MatchProposal) error {
	subject := constants.SubjectMatchAccepted
	// It's crucial that mp.MatchStatus is correctly set by the calling use case.
	// If this is for driver's initial acceptance, it should be MatchStatusPendingCustomerConfirmation.
	log.Printf("Publishing MatchAccept event (status: %s) for MatchID %s (Driver: %s, Passenger: %s) to subject %s",
		mp.MatchStatus, mp.ID, mp.DriverID, mp.PassengerID, subject)
	log.Printf("MatchAccept event details: %+v", mp)

	data, err := json.Marshal(mp)
	if err != nil {
		log.Printf("Error marshalling MatchAccept event for MatchID %s: %v", mp.ID, err)
		return fmt.Errorf("failed to marshal MatchAccept event for match %s: %w", mp.ID, err)
	}

	if err := g.natsClient.Publish(subject, data); err != nil {
		log.Printf("Error publishing MatchAccept event for MatchID %s to NATS subject %s: %v", mp.ID, subject, err)
		return fmt.Errorf("failed to publish MatchAccept event for match %s to subject %s: %w", mp.ID, subject, err)
	}

	log.Printf("Successfully published MatchAccept event for MatchID %s to subject %s", mp.ID, subject)
	return nil
}

// PublishLocationUpdate publishes a location update event to NATS.
func (g *UserGW) PublishLocationUpdate(ctx context.Context, locationEvent *models.LocationUpdate) error {
	subject := constants.SubjectLocationUpdate
	log.Printf("Publishing LocationUpdate event for DriverID %s (RideID: %s) to subject %s",
		locationEvent.DriverID, locationEvent.RideID, subject)
	// %+v might be too verbose for location updates; consider logging specific fields if needed.
	// log.Printf("LocationUpdate event details: %+v", locationEvent)

	data, err := json.Marshal(locationEvent)
	if err != nil {
		log.Printf("Error marshalling LocationUpdate for DriverID %s: %v", locationEvent.DriverID, err)
		return fmt.Errorf("failed to marshal LocationUpdate for DriverID %s: %w", locationEvent.DriverID, err)
	}

	if err := g.natsClient.Publish(subject, data); err != nil {
		log.Printf("Error publishing LocationUpdate for DriverID %s to NATS subject %s: %v", locationEvent.DriverID, subject, err)
		return fmt.Errorf("failed to publish LocationUpdate for DriverID %s to subject %s: %w", locationEvent.DriverID, subject, err)
	}

	// Success log might be too verbose for frequent location updates; consider removing or reducing frequency.
	// log.Printf("Successfully published LocationUpdate for DriverID %s to subject %s", locationEvent.DriverID, subject)
	return nil
}

// PublishRideArrived publishes a ride arrived event to NATS.
func (g *UserGW) PublishRideArrived(ctx context.Context, event *models.RideCompleteEvent) error {
	subject := constants.SubjectRideArrived
	log.Printf("Publishing RideArrived event for RideID %s (MatchID: %s) to subject %s",
		event.RideID, event.MatchID, subject)
	log.Printf("RideArrived event details: %+v", event)

	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("Error marshalling RideArrived event for RideID %s: %v", event.RideID, err)
		return fmt.Errorf("failed to marshal RideArrived event for RideID %s: %w", event.RideID, err)
	}

	if err := g.natsClient.Publish(subject, data); err != nil {
		log.Printf("Error publishing RideArrived event for RideID %s to NATS subject %s: %v", event.RideID, subject, err)
		return fmt.Errorf("failed to publish RideArrived event for RideID %s to subject %s: %w", event.RideID, subject, err)
	}

	log.Printf("Successfully published RideArrived event for RideID %s to subject %s", event.RideID, subject)
	return nil
}

// PublishCustomerConfirmedEvent publishes an event to NATS when a customer confirms a match.
func (g *UserGW) PublishCustomerConfirmedEvent(ctx context.Context, mp models.MatchProposal) error {
	subject := constants.SubjectCustomerMatchConfirmed
	// This status check is good, ensuring the gateway publishes what it's named for.
	if mp.MatchStatus != models.MatchStatusAccepted {
		log.Printf("Error: PublishCustomerConfirmedEvent called with non-Accepted status: %s for MatchID %s", mp.MatchStatus, mp.ID)
		return fmt.Errorf("PublishCustomerConfirmedEvent: invalid match status '%s', expected '%s'", mp.MatchStatus, models.MatchStatusAccepted)
	}

	log.Printf("Publishing CustomerConfirmedEvent for MatchID %s to subject %s", mp.ID, subject)
	log.Printf("CustomerConfirmedEvent details: %+v", mp)

	data, err := json.Marshal(mp)
	if err != nil {
		log.Printf("Error marshalling CustomerConfirmedEvent for MatchID %s: %v", mp.ID, err)
		return fmt.Errorf("failed to marshal CustomerConfirmedEvent for match %s: %w", mp.ID, err)
	}

	if err = g.natsClient.Publish(subject, data); err != nil {
		log.Printf("Error publishing CustomerConfirmedEvent for MatchID %s to NATS subject %s: %v", mp.ID, subject, err)
		return fmt.Errorf("failed to publish CustomerConfirmedEvent for match %s to subject %s: %w", mp.ID, subject, err)
	}
	log.Printf("Successfully published CustomerConfirmedEvent for MatchID %s to subject %s", mp.ID, subject)
	return nil
}

// PublishCustomerRejectedEvent publishes an event to NATS when a customer rejects a match.
func (g *UserGW) PublishCustomerRejectedEvent(ctx context.Context, mp models.MatchProposal) error {
	subject := constants.SubjectCustomerMatchRejected
	if mp.MatchStatus != models.MatchStatusRejected {
		log.Printf("Error: PublishCustomerRejectedEvent called with non-Rejected status: %s for MatchID %s", mp.MatchStatus, mp.ID)
		return fmt.Errorf("PublishCustomerRejectedEvent: invalid match status '%s', expected '%s'", mp.MatchStatus, models.MatchStatusRejected)
	}

	log.Printf("Publishing CustomerRejectedEvent for MatchID %s to subject %s", mp.ID, subject)
	log.Printf("CustomerRejectedEvent details: %+v", mp)

	data, err := json.Marshal(mp)
	if err != nil {
		log.Printf("Error marshalling CustomerRejectedEvent for MatchID %s: %v", mp.ID, err)
		return fmt.Errorf("failed to marshal CustomerRejectedEvent for match %s: %w", mp.ID, err)
	}

	if err = g.natsClient.Publish(subject, data); err != nil {
		log.Printf("Error publishing CustomerRejectedEvent for MatchID %s to NATS subject %s: %v", mp.ID, subject, err)
		return fmt.Errorf("failed to publish CustomerRejectedEvent for match %s to subject %s: %w", mp.ID, subject, err)
	}
	log.Printf("Successfully published CustomerRejectedEvent for MatchID %s to subject %s", mp.ID, subject)
	return nil
}

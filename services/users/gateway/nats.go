package gateway

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// PublishBeaconEvent publishes a beacon event to NATS
func (g *UserGW) PublishBeaconEvent(ctx context.Context, event *models.BeaconEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	fmt.Printf("Publishing beacon event: %s\n", string(data))
	return g.natsClient.Publish(constants.SubjectUserBeacon, data)
}

// PublishMatchAccept publishes a match acceptance event to NATS
func (g *UserGW) MatchAccept(mp *models.MatchProposal) error {
	data, err := json.Marshal(mp)
	if err != nil {
		return err
	}
	fmt.Printf("Publishing match accept: %s\n", string(data))
	return g.natsClient.Publish(constants.SubjectMatchAccepted, data)
}

// PublishLocationUpdate publishes a location update event to NATS
func (g *UserGW) PublishLocationUpdate(ctx context.Context, locationEvent *models.LocationUpdate) error {
	data, err := json.Marshal(locationEvent)
	if err != nil {
		return err
	}
	fmt.Printf("Publishing location update: %s\n", string(data))
	return g.natsClient.Publish(constants.SubjectLocationUpdate, data)
}

func (g *UserGW) PublishRideArrived(ctx context.Context, event *models.RideCompleteEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	fmt.Printf("Publishing ride arrived event: %s\n", string(data))
	return g.natsClient.Publish(constants.SubjectRideArrived, data)
}

// PublishCustomerConfirmedEvent publishes an event to NATS when a customer confirms a match.
func (g *UserGW) PublishCustomerConfirmedEvent(ctx context.Context, mp models.MatchProposal) error {
	if mp.MatchStatus != models.MatchStatusAccepted {
		// It's good practice for the gateway to also be aware of what it's publishing,
		// though the primary check might be in the usecase.
		log.Printf("Error: PublishCustomerConfirmedEvent called with non-Accepted status: %s for MatchID %s", mp.MatchStatus, mp.ID)
		return fmt.Errorf("PublishCustomerConfirmedEvent: invalid match status '%s', expected '%s'", mp.MatchStatus, models.MatchStatusAccepted)
	}

	data, err := json.Marshal(mp)
	if err != nil {
		log.Printf("Error marshalling customer confirmed event for MatchID %s: %v", mp.ID, err)
		return err
	}

	log.Printf("Publishing customer confirmed event for MatchID %s to subject %s: %s", mp.ID, constants.SubjectCustomerMatchConfirmed, string(data))
	err = g.natsClient.Publish(constants.SubjectCustomerMatchConfirmed, data)
	if err != nil {
		log.Printf("Error publishing customer confirmed event for MatchID %s: %v", mp.ID, err)
		return err
	}
	return nil
}

// PublishCustomerRejectedEvent publishes an event to NATS when a customer rejects a match.
func (g *UserGW) PublishCustomerRejectedEvent(ctx context.Context, mp models.MatchProposal) error {
	if mp.MatchStatus != models.MatchStatusRejected {
		log.Printf("Error: PublishCustomerRejectedEvent called with non-Rejected status: %s for MatchID %s", mp.MatchStatus, mp.ID)
		return fmt.Errorf("PublishCustomerRejectedEvent: invalid match status '%s', expected '%s'", mp.MatchStatus, models.MatchStatusRejected)
	}

	data, err := json.Marshal(mp)
	if err != nil {
		log.Printf("Error marshalling customer rejected event for MatchID %s: %v", mp.ID, err)
		return err
	}

	log.Printf("Publishing customer rejected event for MatchID %s to subject %s: %s", mp.ID, constants.SubjectCustomerMatchRejected, string(data))
	err = g.natsClient.Publish(constants.SubjectCustomerMatchRejected, data)
	if err != nil {
		log.Printf("Error publishing customer rejected event for MatchID %s: %v", mp.ID, err)
		return err
	}
	return nil
}

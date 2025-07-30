package nats

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nats-io/nats.go/jetstream"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	"github.com/piresc/nebengjek/services/notification"
)

// NotificationHandler handles JetStream subscriptions for the notification service
type NotificationHandler struct {
	notificationUC notification.NotificationUC
	natsClient     *natspkg.Client
	logger         *slog.Logger
}

// NewNotificationHandler creates a new notification NATS handler
func NewNotificationHandler(notificationUC notification.NotificationUC, client *natspkg.Client, logger *slog.Logger) *NotificationHandler {
	return &NotificationHandler{
		notificationUC: notificationUC,
		natsClient:     client,
		logger:         logger,
	}
}

// InitNATSConsumers initializes all JetStream consumers for the notification service
func (h *NotificationHandler) InitNATSConsumers() error {
	// Define notification-specific consumer configurations
	notificationConsumers := map[string]string{
		"match.found":    "match_proposal_notification",
		"match.accepted": "match_accepted_notification",
		"match.rejected": "match_rejected_notification",
		"ride.pickup":            "ride_pickup_notification",
		"RIDE.started":           "ride_started_notification",
		"RIDE.pickup.arrived":    "ride_pickup_arrived_notification",
		"RIDE.completed":         "ride_completed_notification",
		"RIDE.cancelled":         "ride_cancelled_notification",
		// "PAYMENT.processed": "payment_processed_notification", // TODO: Add when PAYMENT_STREAM is implemented
	}

	// Create consumers for each selective notification event
	for subject, consumerName := range notificationConsumers {
		// Determine the appropriate stream based on subject
		var streamName string
		switch {
		case subject == "match.found" || subject == "match.accepted" || subject == "match.rejected":
			streamName = "MATCH_STREAM"
		case subject == "ride.pickup" || subject == "RIDE.started" || subject == "RIDE.pickup.arrived" || subject == "RIDE.completed" || subject == "RIDE.cancelled":
			streamName = "RIDE_STREAM"
		// case subject == "PAYMENT.processed": // TODO: Add when PAYMENT_STREAM is implemented
		//	streamName = "PAYMENT_STREAM"
		default:
			streamName = "MATCH_STREAM" // fallback
		}

		// Create consumer config using the existing structure
		config := natspkg.ConsumerConfig{
			StreamName:    streamName,
			ConsumerName:  consumerName,
			FilterSubject: subject,
			AckPolicy:     jetstream.AckExplicitPolicy,
			MaxDeliver:    3,
		}

		// Recreate consumer
		if err := h.natsClient.RecreateConsumer(config); err != nil {
			h.logger.Error("Failed to create notification consumer", 
				slog.String("consumer", consumerName),
				slog.String("subject", subject),
				slog.Any("error", err))
			return fmt.Errorf("failed to create consumer %s: %w", consumerName, err)
		}

		// Start consuming messages
		var handler func(jetstream.Msg) error
		switch subject {
		case "match.found":
			handler = h.handleMatchProposalJS
		case "match.accepted":
			handler = h.handleMatchAcceptedJS
		case "match.rejected":
			handler = h.handleMatchRejectedJS
		case "ride.pickup":
			handler = h.handleRidePickupJS
		case "RIDE.started":
			handler = h.handleRideStartedJS
		case "RIDE.pickup.arrived":
			handler = h.handleRidePickupArrivedJS
		case "RIDE.completed":
			handler = h.handleRideCompletedJS
		case "RIDE.cancelled":
			handler = h.handleRideCancelledJS
		// case "PAYMENT.processed": // TODO: Add when PAYMENT_STREAM is implemented
		//	handler = h.handlePaymentProcessedJS
		default:
			h.logger.Warn("No handler defined for subject", slog.String("subject", subject))
			continue
		}

		if err := h.natsClient.ConsumeMessages(streamName, consumerName, func(msg jetstream.Msg) error {
			return handler(msg)
		}); err != nil {
			h.logger.Error("Failed to start consuming messages", 
				slog.String("consumer", consumerName),
				slog.Any("error", err))
			return fmt.Errorf("failed to start consuming for %s: %w", consumerName, err)
		}

		h.logger.Info("Started notification consumer", 
			slog.String("consumer", consumerName),
			slog.String("subject", subject))
	}

	return nil
}

// JetStream handlers that wrap the actual processing methods
func (h *NotificationHandler) handleMatchProposalJS(msg jetstream.Msg) error {
	ctx := context.Background()
	err := h.notificationUC.ProcessMatchProposal(ctx, msg.Data())
	if err != nil {
		h.logger.Error("Failed to process match proposal notification", slog.Any("error", err))
		return err
	}
	return nil // Message is auto-acknowledged by ConsumeMessages on success
}

func (h *NotificationHandler) handleMatchAcceptedJS(msg jetstream.Msg) error {
	ctx := context.Background()
	err := h.notificationUC.ProcessMatchAccepted(ctx, msg.Data())
	if err != nil {
		h.logger.Error("Failed to process match accepted notification", slog.Any("error", err))
		return err
	}
	return nil // Message is auto-acknowledged by ConsumeMessages on success
}

func (h *NotificationHandler) handleMatchRejectedJS(msg jetstream.Msg) error {
	ctx := context.Background()
	err := h.notificationUC.ProcessMatchRejected(ctx, msg.Data())
	if err != nil {
		h.logger.Error("Failed to process match rejected notification", slog.Any("error", err))
		return err
	}
	return nil // Message is auto-acknowledged by ConsumeMessages on success
}

func (h *NotificationHandler) handleRidePickupJS(msg jetstream.Msg) error {
	ctx := context.Background()
	err := h.notificationUC.ProcessRideStarted(ctx, msg.Data())
	if err != nil {
		h.logger.Error("Failed to process ride pickup notification", slog.Any("error", err))
		return err
	}
	return nil // Message is auto-acknowledged by ConsumeMessages on success
}

func (h *NotificationHandler) handleRideStartedJS(msg jetstream.Msg) error {
	ctx := context.Background()
	err := h.notificationUC.ProcessRideStarted(ctx, msg.Data())
	if err != nil {
		h.logger.Error("Failed to process ride started notification", slog.Any("error", err))
		return err
	}
	return nil // Message is auto-acknowledged by ConsumeMessages on success
}

func (h *NotificationHandler) handleRidePickupArrivedJS(msg jetstream.Msg) error {
	ctx := context.Background()
	err := h.notificationUC.ProcessRidePickupArrived(ctx, msg.Data())
	if err != nil {
		h.logger.Error("Failed to process ride pickup arrived notification", slog.Any("error", err))
		return err
	}
	return nil // Message is auto-acknowledged by ConsumeMessages on success
}

func (h *NotificationHandler) handleRideCompletedJS(msg jetstream.Msg) error {
	ctx := context.Background()
	err := h.notificationUC.ProcessRideCompleted(ctx, msg.Data())
	if err != nil {
		h.logger.Error("Failed to process ride completed notification", slog.Any("error", err))
		return err
	}
	return nil // Message is auto-acknowledged by ConsumeMessages on success
}

func (h *NotificationHandler) handleRideCancelledJS(msg jetstream.Msg) error {
	ctx := context.Background()
	err := h.notificationUC.ProcessRideCancelled(ctx, msg.Data())
	if err != nil {
		h.logger.Error("Failed to process ride cancelled notification", slog.Any("error", err))
		return err
	}
	return nil // Message is auto-acknowledged by ConsumeMessages on success
}

func (h *NotificationHandler) handlePaymentProcessedJS(msg jetstream.Msg) error {
	ctx := context.Background()
	err := h.notificationUC.ProcessPaymentProcessed(ctx, msg.Data())
	if err != nil {
		h.logger.Error("Failed to process payment processed notification", slog.Any("error", err))
		return err
	}
	return nil // Message is auto-acknowledged by ConsumeMessages on success
}
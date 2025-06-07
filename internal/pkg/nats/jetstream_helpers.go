package nats

import (
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// StreamConfigBuilder helps build stream configurations
type StreamConfigBuilder struct {
	config StreamConfig
}

// NewStreamConfigBuilder creates a new stream configuration builder
func NewStreamConfigBuilder(name string) *StreamConfigBuilder {
	return &StreamConfigBuilder{
		config: StreamConfig{
			Name:      name,
			Retention: jetstream.LimitsPolicy,
			Storage:   jetstream.FileStorage,
			Replicas:  1,
			MaxAge:    24 * time.Hour,
			MaxBytes:  100 * 1024 * 1024, // 100MB
			MaxMsgs:   1000000,
			Discard:   jetstream.DiscardOld,
		},
	}
}

// WithSubjects sets the subjects for the stream
func (b *StreamConfigBuilder) WithSubjects(subjects ...string) *StreamConfigBuilder {
	b.config.Subjects = subjects
	return b
}

// WithRetention sets the retention policy
func (b *StreamConfigBuilder) WithRetention(retention jetstream.RetentionPolicy) *StreamConfigBuilder {
	b.config.Retention = retention
	return b
}

// WithStorage sets the storage type
func (b *StreamConfigBuilder) WithStorage(storage jetstream.StorageType) *StreamConfigBuilder {
	b.config.Storage = storage
	return b
}

// WithReplicas sets the number of replicas
func (b *StreamConfigBuilder) WithReplicas(replicas int) *StreamConfigBuilder {
	b.config.Replicas = replicas
	return b
}

// WithMaxAge sets the maximum age for messages
func (b *StreamConfigBuilder) WithMaxAge(maxAge time.Duration) *StreamConfigBuilder {
	b.config.MaxAge = maxAge
	return b
}

// WithMaxBytes sets the maximum bytes for the stream
func (b *StreamConfigBuilder) WithMaxBytes(maxBytes int64) *StreamConfigBuilder {
	b.config.MaxBytes = maxBytes
	return b
}

// WithMaxMsgs sets the maximum number of messages
func (b *StreamConfigBuilder) WithMaxMsgs(maxMsgs int64) *StreamConfigBuilder {
	b.config.MaxMsgs = maxMsgs
	return b
}

// WithDiscard sets the discard policy
func (b *StreamConfigBuilder) WithDiscard(discard jetstream.DiscardPolicy) *StreamConfigBuilder {
	b.config.Discard = discard
	return b
}

// Build returns the stream configuration
func (b *StreamConfigBuilder) Build() StreamConfig {
	return b.config
}

// ConsumerConfigBuilder helps build consumer configurations
type ConsumerConfigBuilder struct {
	config ConsumerConfig
}

// NewConsumerConfigBuilder creates a new consumer configuration builder
func NewConsumerConfigBuilder(streamName, consumerName string) *ConsumerConfigBuilder {
	return &ConsumerConfigBuilder{
		config: ConsumerConfig{
			StreamName:    streamName,
			ConsumerName:  consumerName,
			DeliverPolicy: jetstream.DeliverAllPolicy,
			AckPolicy:     jetstream.AckExplicitPolicy,
			AckWait:       30 * time.Second,
			MaxDeliver:    3,
			ReplayPolicy:  jetstream.ReplayInstantPolicy,
			MaxAckPending: 1000,
		},
	}
}

// WithSubject sets the filter subject
func (b *ConsumerConfigBuilder) WithSubject(subject string) *ConsumerConfigBuilder {
	b.config.FilterSubject = subject
	return b
}

// WithDeliverPolicy sets the deliver policy
func (b *ConsumerConfigBuilder) WithDeliverPolicy(policy jetstream.DeliverPolicy) *ConsumerConfigBuilder {
	b.config.DeliverPolicy = policy
	return b
}

// WithAckPolicy sets the acknowledgment policy
func (b *ConsumerConfigBuilder) WithAckPolicy(policy jetstream.AckPolicy) *ConsumerConfigBuilder {
	b.config.AckPolicy = policy
	return b
}

// WithAckWait sets the acknowledgment wait time
func (b *ConsumerConfigBuilder) WithAckWait(ackWait time.Duration) *ConsumerConfigBuilder {
	b.config.AckWait = ackWait
	return b
}

// WithMaxDeliver sets the maximum delivery attempts
func (b *ConsumerConfigBuilder) WithMaxDeliver(maxDeliver int) *ConsumerConfigBuilder {
	b.config.MaxDeliver = maxDeliver
	return b
}

// WithReplayPolicy sets the replay policy
func (b *ConsumerConfigBuilder) WithReplayPolicy(policy jetstream.ReplayPolicy) *ConsumerConfigBuilder {
	b.config.ReplayPolicy = policy
	return b
}

// WithRateLimit sets the rate limit in bytes per second
func (b *ConsumerConfigBuilder) WithRateLimit(rateLimitBps uint64) *ConsumerConfigBuilder {
	b.config.RateLimitBps = rateLimitBps
	return b
}

// WithMaxAckPending sets the maximum pending acknowledgments
func (b *ConsumerConfigBuilder) WithMaxAckPending(maxAckPending int) *ConsumerConfigBuilder {
	b.config.MaxAckPending = maxAckPending
	return b
}

// Build returns the consumer configuration
func (b *ConsumerConfigBuilder) Build() ConsumerConfig {
	return b.config
}

// DefaultStreamConfigs returns the default stream configurations for the ride-sharing system
func DefaultStreamConfigs() []StreamConfig {
	return []StreamConfig{
		NewStreamConfigBuilder("USER_STREAM").
			WithSubjects("user.beacon", "user.finder").
			WithRetention(jetstream.InterestPolicy).
			WithStorage(jetstream.FileStorage).
			WithMaxAge(24 * time.Hour).
			WithMaxBytes(100 * 1024 * 1024).
			WithMaxMsgs(1000000).
			Build(),

		NewStreamConfigBuilder("MATCH_STREAM").
			WithSubjects("match.found", "match.rejected", "match.accepted").
			WithRetention(jetstream.InterestPolicy). // Use InterestPolicy for dual consumption
			WithStorage(jetstream.FileStorage).
			WithMaxAge(1 * time.Hour).
			WithMaxBytes(50 * 1024 * 1024).
			WithMaxMsgs(500000).
			Build(),

		NewStreamConfigBuilder("RIDE_STREAM").
			WithSubjects("ride.pickup", "ride.started", "ride.arrived", "ride.completed").
			WithRetention(jetstream.LimitsPolicy).
			WithStorage(jetstream.FileStorage).
			WithMaxAge(7 * 24 * time.Hour). // 7 days for audit
			WithMaxBytes(200 * 1024 * 1024).
			WithMaxMsgs(2000000).
			Build(),

		NewStreamConfigBuilder("LOCATION_STREAM").
			WithSubjects("location.update", "location.aggregate").
			WithRetention(jetstream.InterestPolicy).
			WithStorage(jetstream.MemoryStorage). // Fast access for location data
			WithMaxAge(2 * time.Hour).
			WithMaxBytes(100 * 1024 * 1024).
			WithMaxMsgs(1000000).
			Build(),
	}
}

// DefaultConsumerConfigs returns common consumer configurations with service-specific naming
func DefaultConsumerConfigs() map[string]ConsumerConfig {
	return map[string]ConsumerConfig{
		// USER_STREAM consumers - user.beacon (dual consumption: users + match)
		"user_beacon_users": NewConsumerConfigBuilder("USER_STREAM", "user_beacon_users").
			WithSubject("user.beacon").
			WithDeliverPolicy(jetstream.DeliverAllPolicy).
			WithAckPolicy(jetstream.AckExplicitPolicy).
			WithMaxDeliver(3).
			Build(),

		"user_beacon_match": NewConsumerConfigBuilder("USER_STREAM", "user_beacon_match").
			WithSubject("user.beacon").
			WithDeliverPolicy(jetstream.DeliverAllPolicy).
			WithAckPolicy(jetstream.AckExplicitPolicy).
			WithMaxDeliver(3).
			Build(),

		// USER_STREAM consumers - user.finder (dual consumption: users + match)
		"user_finder_users": NewConsumerConfigBuilder("USER_STREAM", "user_finder_users").
			WithSubject("user.finder").
			WithDeliverPolicy(jetstream.DeliverAllPolicy).
			WithAckPolicy(jetstream.AckExplicitPolicy).
			WithMaxDeliver(3).
			Build(),

		"user_finder_match": NewConsumerConfigBuilder("USER_STREAM", "user_finder_match").
			WithSubject("user.finder").
			WithDeliverPolicy(jetstream.DeliverAllPolicy).
			WithAckPolicy(jetstream.AckExplicitPolicy).
			WithMaxDeliver(3).
			Build(),

		// MATCH_STREAM consumers - match.found (single consumption: users)
		"match_found_users": NewConsumerConfigBuilder("MATCH_STREAM", "match_found_users").
			WithSubject("match.found").
			WithDeliverPolicy(jetstream.DeliverAllPolicy).
			WithAckPolicy(jetstream.AckExplicitPolicy).
			WithMaxDeliver(5). // Higher retry for critical match events
			Build(),

		// MATCH_STREAM consumers - match.accepted (dual consumption: users + rides)
		"match_accepted_users": NewConsumerConfigBuilder("MATCH_STREAM", "match_accepted_users").
			WithSubject("match.accepted").
			WithDeliverPolicy(jetstream.DeliverAllPolicy).
			WithAckPolicy(jetstream.AckExplicitPolicy).
			WithMaxDeliver(5).
			Build(),

		"match_accepted_rides": NewConsumerConfigBuilder("MATCH_STREAM", "match_accepted_rides").
			WithSubject("match.accepted").
			WithDeliverPolicy(jetstream.DeliverAllPolicy).
			WithAckPolicy(jetstream.AckExplicitPolicy).
			WithMaxDeliver(5).
			Build(),

		// MATCH_STREAM consumers - match.rejected (single consumption: users)
		"match_rejected_users": NewConsumerConfigBuilder("MATCH_STREAM", "match_rejected_users").
			WithSubject("match.rejected").
			WithDeliverPolicy(jetstream.DeliverAllPolicy).
			WithAckPolicy(jetstream.AckExplicitPolicy).
			WithMaxDeliver(3).
			Build(),

		// RIDE_STREAM consumers - ride.pickup (dual consumption: users + match)
		"ride_pickup_users": NewConsumerConfigBuilder("RIDE_STREAM", "ride_pickup_users").
			WithSubject("ride.pickup").
			WithDeliverPolicy(jetstream.DeliverNewPolicy). // FIX: Only process new messages
			WithAckPolicy(jetstream.AckExplicitPolicy).
			WithMaxDeliver(5).
			Build(),

		"ride_pickup_match": NewConsumerConfigBuilder("RIDE_STREAM", "ride_pickup_match").
			WithSubject("ride.pickup").
			WithDeliverPolicy(jetstream.DeliverNewPolicy). // FIX: Only process new messages
			WithAckPolicy(jetstream.AckExplicitPolicy).
			WithMaxDeliver(5).
			Build(),

		// RIDE_STREAM consumers - ride.started (single consumption: users)
		"ride_started_users": NewConsumerConfigBuilder("RIDE_STREAM", "ride_started_users").
			WithSubject("ride.started").
			WithDeliverPolicy(jetstream.DeliverNewPolicy). // FIX: Only process new messages
			WithAckPolicy(jetstream.AckExplicitPolicy).
			WithMaxDeliver(5).
			Build(),

		// RIDE_STREAM consumers - ride.completed (dual consumption: users + match)
		"ride_completed_users": NewConsumerConfigBuilder("RIDE_STREAM", "ride_completed_users").
			WithSubject("ride.completed").
			WithDeliverPolicy(jetstream.DeliverNewPolicy). // FIX: Only process new messages, not old ones
			WithAckPolicy(jetstream.AckExplicitPolicy).
			WithMaxDeliver(3).
			Build(),

		"ride_completed_match": NewConsumerConfigBuilder("RIDE_STREAM", "ride_completed_match").
			WithSubject("ride.completed").
			WithDeliverPolicy(jetstream.DeliverNewPolicy). // FIX: Only process new messages, not old ones
			WithAckPolicy(jetstream.AckExplicitPolicy).
			WithMaxDeliver(3).
			Build(),

		// LOCATION_STREAM consumers - location.update (single consumption: location)
		"location_update_location": NewConsumerConfigBuilder("LOCATION_STREAM", "location_update_location").
			WithSubject("location.update").
			WithDeliverPolicy(jetstream.DeliverNewPolicy). // Only new location updates
			WithAckPolicy(jetstream.AckExplicitPolicy).
			WithMaxDeliver(2). // Fast fail for location updates
			Build(),

		// LOCATION_STREAM consumers - location.aggregate (single consumption: rides)
		"location_aggregate_rides": NewConsumerConfigBuilder("LOCATION_STREAM", "location_aggregate_rides").
			WithSubject("location.aggregate").
			WithDeliverPolicy(jetstream.DeliverAllPolicy).
			WithAckPolicy(jetstream.AckExplicitPolicy).
			WithMaxDeliver(3).
			Build(),
	}
}

// GetStreamForSubject returns the appropriate stream name for a given subject
func GetStreamForSubject(subject string) string {
	switch {
	case subject == "user.beacon" || subject == "user.finder":
		return "USER_STREAM"
	case subject == "match.found" || subject == "match.rejected" || subject == "match.accepted":
		return "MATCH_STREAM"
	case subject == "ride.pickup" || subject == "ride.started" || subject == "ride.arrived" || subject == "ride.completed":
		return "RIDE_STREAM"
	case subject == "location.update" || subject == "location.aggregate":
		return "LOCATION_STREAM"
	default:
		return ""
	}
}

// CreateDefaultConsumersForService creates default consumers for a specific service
func CreateDefaultConsumersForService(client *Client, serviceName string) error {
	configs := DefaultConsumerConfigs()

	var relevantConfigs []ConsumerConfig

	switch serviceName {
	case "users":
		relevantConfigs = append(relevantConfigs,
			configs["user_beacon_users"],
			configs["user_finder_users"],
			configs["match_found_users"],
			configs["match_accepted_users"],
			configs["match_rejected_users"],
			configs["ride_pickup_users"],
			configs["ride_started_users"],
			configs["ride_completed_users"],
		)
	case "match":
		relevantConfigs = append(relevantConfigs,
			configs["user_beacon_match"],
			configs["user_finder_match"],
			configs["ride_pickup_match"],
			configs["ride_completed_match"],
		)
	case "rides":
		relevantConfigs = append(relevantConfigs,
			configs["match_accepted_rides"],
			configs["location_aggregate_rides"],
		)
	case "location":
		relevantConfigs = append(relevantConfigs,
			configs["location_update_location"],
		)
	}

	for _, config := range relevantConfigs {
		if err := client.CreateConsumer(config); err != nil {
			return err
		}
	}

	return nil
}

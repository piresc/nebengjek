package main

import (
	context "context"
	"encoding/json"
	"log"
	"time"

	secretsmanager "cloud.google.com/go/secretmanager/apiv1"
	"github.com/go-redis/redis/v8"
	"github.com/nsqio/go-nsq"
	"github.com/piresc/nebengjek/match-service/domain/errors"
	"github.com/piresc/nebengjek/match-service/proto"
	"github.com/piresc/nebengjek/match-service/repository/redis"
	"github.com/piresc/nebengjek/match-service/usecase"
	"github.com/spf13/viper"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MatchServer struct {
	proto.UnimplementedMatchServiceServer
	producer     *nsq.Producer
	secretClient *secretsmanager.Client
	matchUseCase *usecase.MatchUseCase
}

func NewMatchServer(producer *nsq.Producer, secretClient *secretsmanager.Client) *MatchServer {
	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr: viper.GetString("redis.address"),
		DB:   0,
	})

	// Initialize repositories
	driverRepo := redis.NewDriverRepository(redisClient)
	matchRepo := redis.NewMatchRepository(redisClient)

	// Initialize use case
	matchUseCase := usecase.NewMatchUseCase(driverRepo, matchRepo)

	return &MatchServer{
		producer:     producer,
		secretClient: secretClient,
		matchUseCase: matchUseCase,
	}
}

func (s *MatchServer) RequestMatch(ctx context.Context, req *proto.MatchRequest) (*proto.MatchResponse, error) {
	// Request match through use case
	match, err := s.matchUseCase.RequestMatch(ctx, req.UserId, req.PickupLatitude, req.PickupLongitude, req.DestLatitude, req.DestLongitude)
	if err != nil {
		switch err {
		case errors.ErrInvalidUserID:
			return nil, status.Error(codes.InvalidArgument, err.Error())
		case errors.ErrNoDriversAvailable:
			return nil, status.Error(codes.NotFound, err.Error())
		case errors.ErrInvalidLocation:
			return nil, status.Error(codes.InvalidArgument, err.Error())
		default:
			log.Printf("Error processing match request: %v", err)
			return nil, status.Error(codes.Internal, "failed to process match request")
		}
	}

	// Publish match event to NSQ
	if err := s.publishMatchEvent(match.ID, match.UserID, match.DriverID, match.Status); err != nil {
		log.Printf("Error publishing match event: %v", err)
		return nil, status.Error(codes.Internal, "failed to process match request")
	}

	return &proto.MatchResponse{
		MatchId:    match.ID,
		DriverId:   match.DriverID,
		Status:     match.Status,
		EtaMinutes: match.EtaMinutes,
	}, nil
}

func (s *MatchServer) UpdateDriverLocation(ctx context.Context, req *proto.LocationUpdate) (*proto.LocationUpdateResponse, error) {
	err := s.matchUseCase.UpdateDriverLocation(ctx, req.DriverId, req.Latitude, req.Longitude, req.Status)
	if err != nil {
		switch err {
		case errors.ErrInvalidDriverID:
			return nil, status.Error(codes.InvalidArgument, err.Error())
		case errors.ErrInvalidLocation:
			return nil, status.Error(codes.InvalidArgument, err.Error())
		default:
			log.Printf("Error updating driver location: %v", err)
			return nil, status.Error(codes.Internal, "failed to update driver location")
		}
	}

	return &proto.LocationUpdateResponse{
		Success: true,
		Message: "Location updated successfully",
	}, nil
}

func (s *MatchServer) GetNearbyDrivers(ctx context.Context, req *proto.NearbyDriversRequest) (*proto.NearbyDriversResponse, error) {
	drivers, err := s.matchUseCase.GetNearbyDrivers(ctx, req.Latitude, req.Longitude, req.RadiusKm)
	if err != nil {
		switch err {
		case errors.ErrInvalidLocation:
			return nil, status.Error(codes.InvalidArgument, err.Error())
		default:
			log.Printf("Error finding nearby drivers: %v", err)
			return nil, status.Error(codes.Internal, "failed to find nearby drivers")
		}
	}

	protoDrivers := make([]*proto.Driver, len(drivers))
	for i, driver := range drivers {
		protoDrivers[i] = &proto.Driver{
			DriverId:   driver.ID,
			Latitude:   driver.Latitude,
			Longitude:  driver.Longitude,
			DistanceKm: driver.Distance,
			Status:     driver.Status,
		}
	}

	return &proto.NearbyDriversResponse{
		Drivers: protoDrivers,
	}, nil
}

func (s *MatchServer) publishMatchEvent(matchID, userID, driverID, status string) error {
	message := map[string]interface{}{
		"match_id":  matchID,
		"user_id":   userID,
		"driver_id": driverID,
		"status":    status,
		"timestamp": time.Now().Unix(),
	}

	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	return s.producer.Publish("match_status", data)
}

func generateMatchID() string {
	return time.Now().Format("20060102150405") + "-" + generateRandomString(6)
}

func calculateETA(distanceKm float64) float64 {
	// Simplified ETA calculation (assuming average speed of 30 km/h)
	return (distanceKm / 30.0) * 60.0
}

func generateRandomString(n int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[time.Now().UnixNano()%int64(len(letterBytes))]
	}
	return string(b)
}

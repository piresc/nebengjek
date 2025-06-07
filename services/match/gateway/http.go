package gateway

import (
	"context"
	"fmt"
	"log/slog"

	httpclient "github.com/piresc/nebengjek/internal/pkg/http"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/internal/pkg/observability"
)

// HTTPGateway wraps the location client for HTTP operations
type HTTPGateway struct {
	locationClient *LocationClient
}

// LocationClient is a simplified HTTP client for communicating with the location service
type LocationClient struct {
	client  *httpclient.Client
	tracer  observability.Tracer
	logger  *slog.Logger
	baseURL string
}

// NewHTTPGateway creates a new HTTP gateway with location client
func NewHTTPGateway(locationServiceURL string, config *models.APIKeyConfig, tracer observability.Tracer, logger *slog.Logger) *HTTPGateway {
	locationClient := &LocationClient{
		client: httpclient.NewClient(httpclient.Config{
			APIKey:  config.MatchService,
			BaseURL: locationServiceURL,
			Timeout: 30 * 1000000000, // 30 seconds in nanoseconds
		}),
		tracer:  tracer,
		logger:  logger,
		baseURL: locationServiceURL,
	}
	return &HTTPGateway{
		locationClient: locationClient,
	}
}

// AddAvailableDriver adds a driver to the available drivers geo set via HTTP
func (gw *LocationClient) AddAvailableDriver(ctx context.Context, driverID string, location *models.Location) error {
	endpoint := fmt.Sprintf("/internal/drivers/%s/available", driverID)

	// Start APM segment if tracer is available
	var endSegment func()
	if gw.tracer != nil {
		ctx, endSegment = gw.tracer.StartSegment(ctx, "External/location-service/add-driver")
		defer endSegment()
	}

	request := map[string]interface{}{
		"location": location,
	}

	var response map[string]string
	err := gw.client.PostJSON(ctx, endpoint, request, &response)
	if err != nil {
		if gw.logger != nil {
			gw.logger.Error("Failed to add available driver",
				slog.String("driver_id", driverID),
				slog.Any("error", err))
		}
		return fmt.Errorf("failed to add available driver: %w", err)
	}

	return nil
}

// RemoveAvailableDriver removes a driver from the available drivers sets via HTTP
func (gw *LocationClient) RemoveAvailableDriver(ctx context.Context, driverID string) error {
	endpoint := fmt.Sprintf("/internal/drivers/%s/available", driverID)

	// Start APM segment if tracer is available
	var endSegment func()
	if gw.tracer != nil {
		ctx, endSegment = gw.tracer.StartSegment(ctx, "External/location-service/remove-driver")
		defer endSegment()
	}

	resp, err := gw.client.Delete(ctx, endpoint)
	if err != nil {
		if gw.logger != nil {
			gw.logger.Error("Failed to remove available driver",
				slog.String("driver_id", driverID),
				slog.Any("error", err))
		}
		return fmt.Errorf("failed to remove available driver: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		err = fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
		if gw.logger != nil {
			gw.logger.Error("Failed to remove available driver",
				slog.String("driver_id", driverID),
				slog.Any("error", err))
		}
		return fmt.Errorf("failed to remove available driver: %w", err)
	}

	return nil
}

// AddAvailablePassenger adds a passenger to the Redis geospatial index via HTTP
func (gw *LocationClient) AddAvailablePassenger(ctx context.Context, passengerID string, location *models.Location) error {
	endpoint := fmt.Sprintf("/internal/passengers/%s/available", passengerID)

	// Start APM segment if tracer is available
	var endSegment func()
	if gw.tracer != nil {
		ctx, endSegment = gw.tracer.StartSegment(ctx, "External/location-service/add-passenger")
		defer endSegment()
	}

	request := map[string]interface{}{
		"location": location,
	}

	var response map[string]string
	err := gw.client.PostJSON(ctx, endpoint, request, &response)
	if err != nil {
		if gw.logger != nil {
			gw.logger.Error("Failed to add available passenger",
				slog.String("passenger_id", passengerID),
				slog.Any("error", err))
		}
		return fmt.Errorf("failed to add available passenger: %w", err)
	}

	return nil
}

// RemoveAvailablePassenger removes a passenger from the Redis geospatial index via HTTP
func (gw *LocationClient) RemoveAvailablePassenger(ctx context.Context, passengerID string) error {
	endpoint := fmt.Sprintf("/internal/passengers/%s/available", passengerID)

	// Start APM segment if tracer is available
	var endSegment func()
	if gw.tracer != nil {
		ctx, endSegment = gw.tracer.StartSegment(ctx, "External/location-service/remove-passenger")
		defer endSegment()
	}

	resp, err := gw.client.Delete(ctx, endpoint)
	if err != nil {
		if gw.logger != nil {
			gw.logger.Error("Failed to remove available passenger",
				slog.String("passenger_id", passengerID),
				slog.Any("error", err))
		}
		return fmt.Errorf("failed to remove available passenger: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		err = fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
		if gw.logger != nil {
			gw.logger.Error("Failed to remove available passenger",
				slog.String("passenger_id", passengerID),
				slog.Any("error", err))
		}
		return fmt.Errorf("failed to remove available passenger: %w", err)
	}

	return nil
}

// FindNearbyDrivers finds available drivers within the specified radius via HTTP
func (gw *LocationClient) FindNearbyDrivers(ctx context.Context, location *models.Location, radiusKm float64) ([]*models.NearbyUser, error) {
	endpoint := fmt.Sprintf("/internal/drivers/nearby?lat=%f&lng=%f&radius=%f",
		location.Latitude, location.Longitude, radiusKm)

	// Start APM segment if tracer is available
	var endSegment func()
	if gw.tracer != nil {
		ctx, endSegment = gw.tracer.StartSegment(ctx, "External/location-service/find-drivers")
		defer endSegment()
	}

	var nearbyDrivers []*models.NearbyUser
	err := gw.client.GetJSON(ctx, endpoint, &nearbyDrivers)
	if err != nil {
		if gw.logger != nil {
			gw.logger.Error("Failed to find nearby drivers", slog.Any("error", err))
		}
		return nil, fmt.Errorf("failed to find nearby drivers: %w", err)
	}

	return nearbyDrivers, nil
}

// GetDriverLocation retrieves a driver's last known location via HTTP
func (gw *LocationClient) GetDriverLocation(ctx context.Context, driverID string) (models.Location, error) {
	endpoint := fmt.Sprintf("/internal/drivers/%s/location", driverID)

	// Start APM segment if tracer is available
	var endSegment func()
	if gw.tracer != nil {
		ctx, endSegment = gw.tracer.StartSegment(ctx, "External/location-service/get-driver-location")
		defer endSegment()
	}

	var location models.Location
	err := gw.client.GetJSON(ctx, endpoint, &location)
	if err != nil {
		if gw.logger != nil {
			gw.logger.Error("Failed to get driver location",
				slog.String("driver_id", driverID),
				slog.Any("error", err))
		}
		return models.Location{}, fmt.Errorf("failed to get driver location: %w", err)
	}

	return location, nil
}

// GetPassengerLocation retrieves a passenger's last known location via HTTP
func (gw *LocationClient) GetPassengerLocation(ctx context.Context, passengerID string) (models.Location, error) {
	endpoint := fmt.Sprintf("/internal/passengers/%s/location", passengerID)

	// Start APM segment if tracer is available
	var endSegment func()
	if gw.tracer != nil {
		ctx, endSegment = gw.tracer.StartSegment(ctx, "External/location-service/get-passenger-location")
		defer endSegment()
	}

	var location models.Location
	err := gw.client.GetJSON(ctx, endpoint, &location)
	if err != nil {
		if gw.logger != nil {
			gw.logger.Error("Failed to get passenger location",
				slog.String("passenger_id", passengerID),
				slog.Any("error", err))
		}
		return models.Location{}, fmt.Errorf("failed to get passenger location: %w", err)
	}

	return location, nil
}

// HTTPGateway delegation methods

// AddAvailableDriver delegates to the location client
func (gw *HTTPGateway) AddAvailableDriver(ctx context.Context, driverID string, location *models.Location) error {
	return gw.locationClient.AddAvailableDriver(ctx, driverID, location)
}

// RemoveAvailableDriver delegates to the location client
func (gw *HTTPGateway) RemoveAvailableDriver(ctx context.Context, driverID string) error {
	return gw.locationClient.RemoveAvailableDriver(ctx, driverID)
}

// AddAvailablePassenger delegates to the location client
func (gw *HTTPGateway) AddAvailablePassenger(ctx context.Context, passengerID string, location *models.Location) error {
	return gw.locationClient.AddAvailablePassenger(ctx, passengerID, location)
}

// RemoveAvailablePassenger delegates to the location client
func (gw *HTTPGateway) RemoveAvailablePassenger(ctx context.Context, passengerID string) error {
	return gw.locationClient.RemoveAvailablePassenger(ctx, passengerID)
}

// FindNearbyDrivers delegates to the location client
func (gw *HTTPGateway) FindNearbyDrivers(ctx context.Context, location *models.Location, radiusKm float64) ([]*models.NearbyUser, error) {
	return gw.locationClient.FindNearbyDrivers(ctx, location, radiusKm)
}

// GetDriverLocation delegates to the location client
func (gw *HTTPGateway) GetDriverLocation(ctx context.Context, driverID string) (models.Location, error) {
	return gw.locationClient.GetDriverLocation(ctx, driverID)
}

// GetPassengerLocation delegates to the location client
func (gw *HTTPGateway) GetPassengerLocation(ctx context.Context, passengerID string) (models.Location, error) {
	return gw.locationClient.GetPassengerLocation(ctx, passengerID)
}

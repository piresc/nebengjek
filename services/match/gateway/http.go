package gateway

import (
	"context"
	"fmt"

	httpclient "github.com/piresc/nebengjek/internal/pkg/http"
	"github.com/piresc/nebengjek/internal/pkg/logger"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// HTTPGateway wraps the location client for HTTP operations
type HTTPGateway struct {
	locationClient *LocationClient
}

// LocationClient is an enhanced HTTP client for communicating with the location service
type LocationClient struct {
	apiClient *httpclient.APIKeyClient
	baseURL   string
}

// NewHTTPGateway creates a new HTTP gateway with location client
func NewHTTPGateway(locationServiceURL string, config *models.APIKeyConfig) *HTTPGateway {
	locationClient := &LocationClient{
		apiClient: httpclient.NewAPIKeyClient(config, "match-service", locationServiceURL),
		baseURL:   locationServiceURL,
	}
	return &HTTPGateway{
		locationClient: locationClient,
	}
}

// AddAvailableDriver adds a driver to the available drivers geo set via HTTP
func (gw *LocationClient) AddAvailableDriver(ctx context.Context, driverID string, location *models.Location) error {
	endpoint := fmt.Sprintf("/internal/drivers/%s/available", driverID)

	request := map[string]interface{}{
		"location": location,
	}

	var response map[string]string
	err := gw.apiClient.PostJSON(ctx, endpoint, request, &response)
	if err != nil {
		logger.Error("Failed to add available driver",
			logger.String("driver_id", driverID),
			logger.ErrorField(err))
		return fmt.Errorf("failed to add available driver: %w", err)
	}

	return nil
}

// RemoveAvailableDriver removes a driver from the available drivers sets via HTTP
func (gw *LocationClient) RemoveAvailableDriver(ctx context.Context, driverID string) error {
	endpoint := fmt.Sprintf("/internal/drivers/%s/available", driverID)

	resp, err := gw.apiClient.Delete(ctx, endpoint)
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			err = fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
		}
	}
	if err != nil {
		logger.Error("Failed to remove available driver",
			logger.String("driver_id", driverID),
			logger.ErrorField(err))
		return fmt.Errorf("failed to remove available driver: %w", err)
	}

	return nil
}

// AddAvailablePassenger adds a passenger to the Redis geospatial index via HTTP
func (gw *LocationClient) AddAvailablePassenger(ctx context.Context, passengerID string, location *models.Location) error {
	endpoint := fmt.Sprintf("/internal/passengers/%s/available", passengerID)

	request := map[string]interface{}{
		"location": location,
	}

	var response map[string]string
	err := gw.apiClient.PostJSON(ctx, endpoint, request, &response)
	if err != nil {
		logger.Error("Failed to add available passenger",
			logger.String("passenger_id", passengerID),
			logger.ErrorField(err))
		return fmt.Errorf("failed to add available passenger: %w", err)
	}

	return nil
}

// RemoveAvailablePassenger removes a passenger from the Redis geospatial index via HTTP
func (gw *LocationClient) RemoveAvailablePassenger(ctx context.Context, passengerID string) error {
	endpoint := fmt.Sprintf("/internal/passengers/%s/available", passengerID)

	resp, err := gw.apiClient.Delete(ctx, endpoint)
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			err = fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
		}
	}
	if err != nil {
		logger.Error("Failed to remove available passenger",
			logger.String("passenger_id", passengerID),
			logger.ErrorField(err))
		return fmt.Errorf("failed to remove available passenger: %w", err)
	}

	return nil
}

// FindNearbyDrivers finds available drivers within the specified radius via HTTP
func (gw *LocationClient) FindNearbyDrivers(ctx context.Context, location *models.Location, radiusKm float64) ([]*models.NearbyUser, error) {
	endpoint := fmt.Sprintf("/internal/drivers/nearby?lat=%f&lng=%f&radius=%f",
		location.Latitude, location.Longitude, radiusKm)

	var nearbyDrivers []*models.NearbyUser
	err := gw.apiClient.GetJSON(ctx, endpoint, &nearbyDrivers)
	if err != nil {
		logger.Error("Failed to find nearby drivers", logger.ErrorField(err))
		return nil, fmt.Errorf("failed to find nearby drivers: %w", err)
	}

	return nearbyDrivers, nil
}

// GetDriverLocation retrieves a driver's last known location via HTTP
func (gw *LocationClient) GetDriverLocation(ctx context.Context, driverID string) (models.Location, error) {
	endpoint := fmt.Sprintf("/internal/drivers/%s/location", driverID)

	var location models.Location
	err := gw.apiClient.GetJSON(ctx, endpoint, &location)
	if err != nil {
		logger.Error("Failed to get driver location",
			logger.String("driver_id", driverID),
			logger.ErrorField(err))
		return models.Location{}, fmt.Errorf("failed to get driver location: %w", err)
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

// GetPassengerLocation retrieves a passenger's last known location via HTTP
func (gw *LocationClient) GetPassengerLocation(ctx context.Context, passengerID string) (models.Location, error) {
	endpoint := fmt.Sprintf("/internal/passengers/%s/location", passengerID)

	var location models.Location
	err := gw.apiClient.GetJSON(ctx, endpoint, &location)
	if err != nil {
		logger.Error("Failed to get passenger location",
			logger.String("passenger_id", passengerID),
			logger.ErrorField(err))
		return models.Location{}, fmt.Errorf("failed to get passenger location: %w", err)
	}

	return location, nil
}

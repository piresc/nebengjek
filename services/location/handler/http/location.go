package http

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/logger"
	"github.com/piresc/nebengjek/internal/pkg/models"
	nrpkg "github.com/piresc/nebengjek/internal/pkg/newrelic"
	"github.com/piresc/nebengjek/internal/utils"
	"github.com/piresc/nebengjek/services/location"
)

// LocationHandler handles HTTP requests for location operations
type LocationHandler struct {
	locationUC location.LocationUC
}

// NewLocationHandler creates a new location HTTP handler
func NewLocationHandler(locationUC location.LocationUC) *LocationHandler {
	return &LocationHandler{
		locationUC: locationUC,
	}
}

// AddAvailableDriver adds a driver to the available pool
func (h *LocationHandler) AddAvailableDriver(c echo.Context) error {
	// Get transaction from Echo context using centralized package
	txn := nrpkg.FromEchoContext(c)
	nrpkg.SetTransactionName(txn, "Location.AddAvailableDriver")

	driverID := c.Param("id")
	if driverID == "" {
		return utils.BadRequestResponse(c, "driver_id is required")
	}

	nrpkg.AddTransactionAttribute(txn, "endpoint", "add_available_driver")
	nrpkg.AddTransactionAttribute(txn, "driver.id", driverID)

	var req struct {
		Location models.Location `json:"location"`
	}

	if err := c.Bind(&req); err != nil {
		nrpkg.NoticeTransactionError(txn, err)
		logger.Error("Failed to bind request", logger.ErrorField(err))
		return utils.BadRequestResponse(c, "invalid request body")
	}

	if err := h.locationUC.AddAvailableDriver(c.Request().Context(), driverID, &req.Location); err != nil {
		nrpkg.NoticeTransactionError(txn, err)
		logger.Error("Failed to add available driver",
			logger.String("driver_id", driverID),
			logger.ErrorField(err))
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, "failed to add driver")
	}

	return utils.SuccessResponse(c, http.StatusOK, "Driver added successfully", map[string]string{"status": "success"})
}

// RemoveAvailableDriver removes a driver from the available pool
func (h *LocationHandler) RemoveAvailableDriver(c echo.Context) error {
	// Get transaction from Echo context using centralized package
	txn := nrpkg.FromEchoContext(c)
	nrpkg.SetTransactionName(txn, "Location.RemoveAvailableDriver")

	driverID := c.Param("id")
	if driverID == "" {
		return utils.BadRequestResponse(c, "driver_id is required")
	}

	nrpkg.AddTransactionAttribute(txn, "endpoint", "remove_available_driver")
	nrpkg.AddTransactionAttribute(txn, "driver.id", driverID)

	if err := h.locationUC.RemoveAvailableDriver(c.Request().Context(), driverID); err != nil {
		nrpkg.NoticeTransactionError(txn, err)
		logger.Error("Failed to remove available driver",
			logger.String("driver_id", driverID),
			logger.ErrorField(err))
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, "failed to remove driver")
	}

	return utils.SuccessResponse(c, http.StatusOK, "Driver removed successfully", map[string]string{"status": "success"})
}

// AddAvailablePassenger adds a passenger to the available pool
func (h *LocationHandler) AddAvailablePassenger(c echo.Context) error {
	// Get transaction from Echo context using centralized package
	txn := nrpkg.FromEchoContext(c)
	nrpkg.SetTransactionName(txn, "Location.AddAvailablePassenger")

	passengerID := c.Param("id")
	if passengerID == "" {
		return utils.BadRequestResponse(c, "passenger_id is required")
	}

	nrpkg.AddTransactionAttribute(txn, "endpoint", "add_available_passenger")
	nrpkg.AddTransactionAttribute(txn, "passenger.id", passengerID)

	var req struct {
		Location models.Location `json:"location"`
	}

	if err := c.Bind(&req); err != nil {
		nrpkg.NoticeTransactionError(txn, err)
		logger.Error("Failed to bind request", logger.ErrorField(err))
		return utils.BadRequestResponse(c, "invalid request body")
	}

	if err := h.locationUC.AddAvailablePassenger(c.Request().Context(), passengerID, &req.Location); err != nil {
		nrpkg.NoticeTransactionError(txn, err)
		logger.Error("Failed to add available passenger",
			logger.String("passenger_id", passengerID),
			logger.ErrorField(err))
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, "failed to add passenger")
	}

	return utils.SuccessResponse(c, http.StatusOK, "Passenger added successfully", map[string]string{"status": "success"})
}

// RemoveAvailablePassenger removes a passenger from the available pool
func (h *LocationHandler) RemoveAvailablePassenger(c echo.Context) error {
	// Get transaction from Echo context using centralized package
	txn := nrpkg.FromEchoContext(c)
	nrpkg.SetTransactionName(txn, "Location.RemoveAvailablePassenger")

	passengerID := c.Param("id")
	if passengerID == "" {
		return utils.BadRequestResponse(c, "passenger_id is required")
	}

	nrpkg.AddTransactionAttribute(txn, "endpoint", "remove_available_passenger")
	nrpkg.AddTransactionAttribute(txn, "passenger.id", passengerID)

	if err := h.locationUC.RemoveAvailablePassenger(c.Request().Context(), passengerID); err != nil {
		nrpkg.NoticeTransactionError(txn, err)
		logger.Error("Failed to remove available passenger",
			logger.String("passenger_id", passengerID),
			logger.ErrorField(err))
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, "failed to remove passenger")
	}

	return utils.SuccessResponse(c, http.StatusOK, "Passenger removed successfully", map[string]string{"status": "success"})
}

// FindNearbyDrivers finds drivers near a location
func (h *LocationHandler) FindNearbyDrivers(c echo.Context) error {
	// Get transaction from Echo context using centralized package
	txn := nrpkg.FromEchoContext(c)
	nrpkg.SetTransactionName(txn, "Location.FindNearbyDrivers")

	latStr := c.QueryParam("lat")
	lngStr := c.QueryParam("lng")
	radiusStr := c.QueryParam("radius")

	if latStr == "" || lngStr == "" || radiusStr == "" {
		return utils.BadRequestResponse(c, "lat, lng, and radius are required")
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		return utils.BadRequestResponse(c, "invalid latitude")
	}

	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil {
		return utils.BadRequestResponse(c, "invalid longitude")
	}

	radius, err := strconv.ParseFloat(radiusStr, 64)
	if err != nil {
		return utils.BadRequestResponse(c, "invalid radius")
	}

	location := &models.Location{
		Latitude:  lat,
		Longitude: lng,
	}

	nrpkg.AddTransactionAttribute(txn, "endpoint", "find_nearby_drivers")
	nrpkg.AddTransactionAttribute(txn, "location.latitude", lat)
	nrpkg.AddTransactionAttribute(txn, "location.longitude", lng)
	nrpkg.AddTransactionAttribute(txn, "search.radius", radius)

	drivers, err := h.locationUC.FindNearbyDrivers(c.Request().Context(), location, radius)
	if err != nil {
		nrpkg.NoticeTransactionError(txn, err)
		logger.Error("Failed to find nearby drivers", logger.ErrorField(err))
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, "failed to find drivers")
	}

	return utils.SuccessResponse(c, http.StatusOK, "Nearby drivers found", drivers)
}

// GetDriverLocation gets a driver's location
func (h *LocationHandler) GetDriverLocation(c echo.Context) error {
	// Get transaction from Echo context using centralized package
	txn := nrpkg.FromEchoContext(c)
	nrpkg.SetTransactionName(txn, "Location.GetDriverLocation")

	driverID := c.Param("id")
	if driverID == "" {
		return utils.BadRequestResponse(c, "driver_id is required")
	}

	nrpkg.AddTransactionAttribute(txn, "endpoint", "get_driver_location")
	nrpkg.AddTransactionAttribute(txn, "driver.id", driverID)

	location, err := h.locationUC.GetDriverLocation(c.Request().Context(), driverID)
	if err != nil {
		nrpkg.NoticeTransactionError(txn, err)
		logger.Error("Failed to get driver location",
			logger.String("driver_id", driverID),
			logger.ErrorField(err))
		return utils.NotFoundResponse(c, "driver location not found")
	}

	return utils.SuccessResponse(c, http.StatusOK, "Driver location retrieved", location)
}

// GetPassengerLocation gets a passenger's location
func (h *LocationHandler) GetPassengerLocation(c echo.Context) error {
	// Get transaction from Echo context using centralized package
	txn := nrpkg.FromEchoContext(c)
	nrpkg.SetTransactionName(txn, "Location.GetPassengerLocation")

	passengerID := c.Param("id")
	if passengerID == "" {
		return utils.BadRequestResponse(c, "passenger_id is required")
	}

	nrpkg.AddTransactionAttribute(txn, "endpoint", "get_passenger_location")
	nrpkg.AddTransactionAttribute(txn, "passenger.id", passengerID)

	location, err := h.locationUC.GetPassengerLocation(c.Request().Context(), passengerID)
	if err != nil {
		nrpkg.NoticeTransactionError(txn, err)
		logger.Error("Failed to get passenger location",
			logger.String("passenger_id", passengerID),
			logger.ErrorField(err))
		return utils.NotFoundResponse(c, "passenger location not found")
	}

	return utils.SuccessResponse(c, http.StatusOK, "Passenger location retrieved", location)
}

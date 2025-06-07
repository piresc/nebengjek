package http

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/logger"
	"github.com/piresc/nebengjek/internal/pkg/models"
	nrpkg "github.com/piresc/nebengjek/internal/pkg/newrelic"
	"github.com/piresc/nebengjek/internal/utils"
	"github.com/piresc/nebengjek/services/rides"
)

// RidesHandler handles HTTP requests for ride operations
type RidesHandler struct {
	rideUC rides.RideUC
}

// NewRidesHandler creates a new ride HTTP handler
func NewRidesHandler(rideUC rides.RideUC) *RidesHandler {
	return &RidesHandler{
		rideUC: rideUC,
	}
}

// StartRide handles the start trip request for a ride
func (h *RidesHandler) StartRide(c echo.Context) error {
	// Get transaction from Echo context using centralized package
	txn := nrpkg.FromEchoContext(c)
	nrpkg.SetTransactionName(txn, "Rides.StartRide")

	rideID := c.Param("rideID")
	if rideID == "" {
		return utils.BadRequestResponse(c, "Ride ID is required")
	}

	var req models.RideStartRequest
	if err := c.Bind(&req); err != nil {
		nrpkg.NoticeTransactionError(txn, err)
		return utils.BadRequestResponse(c, "Invalid request body: "+err.Error())
	}

	req.RideID = rideID

	// Log the incoming request for debugging
	logger.Info("Received start ride request",
		logger.String("ride_id", rideID),
		logger.String("client_ip", c.RealIP()),
		logger.String("user_agent", c.Request().UserAgent()),
		logger.Any("driver_location", req.DriverLocation),
		logger.Any("passenger_location", req.PassengerLocation))

	if req.DriverLocation == nil || req.PassengerLocation == nil {
		logger.Error("Missing required location data",
			logger.String("ride_id", rideID),
			logger.Bool("has_driver_location", req.DriverLocation != nil),
			logger.Bool("has_passenger_location", req.PassengerLocation != nil))
		return utils.BadRequestResponse(c, "Driver and passenger locations are required")
	}

	resp, err := h.rideUC.StartRide(c.Request().Context(), req)
	if err != nil {
		logger.Error("Failed to start ride in handler",
			logger.String("ride_id", rideID),
			logger.ErrorField(err))
		nrpkg.NoticeTransactionError(txn, err)
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, "Failed to start trip: "+err.Error())
	}

	logger.Info("Successfully started ride",
		logger.String("ride_id", rideID),
		logger.String("new_status", string(resp.Status)))

	return utils.SuccessResponse(c, http.StatusOK, "Trip started successfully", resp)
}

// RideArrived handles the ride arrival notification
func (h *RidesHandler) RideArrived(c echo.Context) error {
	// Get transaction from Echo context using centralized package
	txn := nrpkg.FromEchoContext(c)
	nrpkg.SetTransactionName(txn, "Rides.RideArrived")

	rideID := c.Param("rideID")
	if rideID == "" {
		return utils.BadRequestResponse(c, "Ride ID is required")
	}

	nrpkg.AddTransactionAttribute(txn, "endpoint", "ride_arrived")
	nrpkg.AddTransactionAttribute(txn, "ride.id", rideID)

	var req models.RideArrivalReq
	if err := c.Bind(&req); err != nil {
		nrpkg.NoticeTransactionError(txn, err)
		return utils.BadRequestResponse(c, "Invalid request body: "+err.Error())
	}

	paymentReq, err := h.rideUC.RideArrived(c.Request().Context(), req)
	if err != nil {
		nrpkg.NoticeTransactionError(txn, err)
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, "Failed to process ride arrival: "+err.Error())
	}

	return utils.SuccessResponse(c, http.StatusOK, "Ride arrived successfully", paymentReq)
}

// ProcessPayment handles the payment processing for a completed ride
func (h *RidesHandler) ProcessPayment(c echo.Context) error {
	// Get transaction from Echo context using centralized package
	txn := nrpkg.FromEchoContext(c)
	nrpkg.SetTransactionName(txn, "Rides.ProcessPayment")

	rideID := c.Param("rideID")
	if rideID == "" {
		return utils.BadRequestResponse(c, "Ride ID is required")
	}

	nrpkg.AddTransactionAttribute(txn, "endpoint", "process_payment")
	nrpkg.AddTransactionAttribute(txn, "ride.id", rideID)

	var req models.PaymentProccessRequest
	if err := c.Bind(&req); err != nil {
		nrpkg.NoticeTransactionError(txn, err)
		return utils.BadRequestResponse(c, "Invalid request body: "+err.Error())
	}

	req.RideID = rideID

	payment, err := h.rideUC.ProcessPayment(c.Request().Context(), req)
	if err != nil {
		nrpkg.NoticeTransactionError(txn, err)
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, "Failed to process payment: "+err.Error())
	}

	return utils.SuccessResponse(c, http.StatusOK, "Payment processed successfully", payment)
}

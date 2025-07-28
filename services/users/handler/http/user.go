package http

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/models"
	nrpkg "github.com/piresc/nebengjek/internal/pkg/newrelic"
	"github.com/piresc/nebengjek/internal/utils"
	"github.com/piresc/nebengjek/services/users"
)

// UserHandler handles HTTP requests for user operations
type UserHandler struct {
	userUC users.UserUC
}

// NewUserHandler creates a new user handler
func NewUserHandler(
	userUC users.UserUC,
) *UserHandler {
	return &UserHandler{
		userUC: userUC,
	}
}

// CreateUser handles user creation requests
func (h *UserHandler) CreateUser(c echo.Context) error {
	// Get transaction from Echo context using centralized package
	txn := nrpkg.FromEchoContext(c)
	nrpkg.SetTransactionName(txn, "CreateUser")

	var user models.User
	if err := c.Bind(&user); err != nil {
		nrpkg.NoticeTransactionError(txn, err)
		return utils.BadRequestResponse(c, "Invalid request payload")
	}

	nrpkg.AddTransactionAttribute(txn, "user.msisdn", user.MSISDN)
	nrpkg.AddTransactionAttribute(txn, "user.role", user.Role)

	err := h.userUC.RegisterUser(c.Request().Context(), &user)
	if err != nil {
		nrpkg.NoticeTransactionError(txn, err)
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, "Failed to create user")
	}

	return utils.SuccessResponse(c, http.StatusCreated, "User created successfully", user)
}

// GetUser handles user retrieval requests
func (h *UserHandler) GetUser(c echo.Context) error {
	// Get transaction from Echo context using centralized package
	txn := nrpkg.FromEchoContext(c)
	nrpkg.SetTransactionName(txn, "GetUser")

	userID := c.Param("id")
	if userID == "" {
		return utils.BadRequestResponse(c, "Invalid user ID")
	}

	nrpkg.AddTransactionAttribute(txn, "user.id", userID)

	user, err := h.userUC.GetUserByID(c.Request().Context(), userID)
	if err != nil {
		nrpkg.NoticeTransactionError(txn, err)
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, "Failed to retrieve user")
	}

	return utils.SuccessResponse(c, http.StatusOK, "User retrieved successfully", user)
}

// RegisterDriver handles driver registration requests
func (h *UserHandler) RegisterDriver(c echo.Context) error {
	// Get transaction from Echo context using centralized package
	txn := nrpkg.FromEchoContext(c)
	nrpkg.SetTransactionName(txn, "RegisterDriver")

	var user models.User
	if err := c.Bind(&user); err != nil {
		nrpkg.NoticeTransactionError(txn, err)
		return utils.BadRequestResponse(c, "Invalid request payload")
	}

	nrpkg.AddTransactionAttribute(txn, "user.msisdn", user.MSISDN)
	nrpkg.AddTransactionAttribute(txn, "user.role", "driver")

	err := h.userUC.RegisterDriver(c.Request().Context(), &user)
	if err != nil {
		nrpkg.NoticeTransactionError(txn, err)
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, "Failed to register driver")
	}

	return utils.SuccessResponse(c, http.StatusCreated, "Driver registered successfully", user)
}

// UpdateFinderStatus handles finder status update requests
func (h *UserHandler) UpdateFinderStatus(c echo.Context) error {
	// Get transaction from Echo context using centralized package
	txn := nrpkg.FromEchoContext(c)
	nrpkg.SetTransactionName(txn, "UpdateFinderStatus")

	var finderReq models.FinderRequest
	if err := c.Bind(&finderReq); err != nil {
		nrpkg.NoticeTransactionError(txn, err)
		return utils.BadRequestResponse(c, "Invalid request payload")
	}

	nrpkg.AddTransactionAttribute(txn, "user.msisdn", finderReq.MSISDN)
	nrpkg.AddTransactionAttribute(txn, "finder.is_active", finderReq.IsActive)

	err := h.userUC.UpdateFinderStatus(c.Request().Context(), &finderReq)
	if err != nil {
		nrpkg.NoticeTransactionError(txn, err)
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, "Failed to update finder status")
	}

	return utils.SuccessResponse(c, http.StatusOK, "Finder status updated successfully", nil)
}

// UpdateBeaconStatus handles beacon status update requests
func (h *UserHandler) UpdateBeaconStatus(c echo.Context) error {
	// Get transaction from Echo context using centralized package
	txn := nrpkg.FromEchoContext(c)
	nrpkg.SetTransactionName(txn, "UpdateBeaconStatus")

	var beaconReq models.BeaconRequest
	if err := c.Bind(&beaconReq); err != nil {
		nrpkg.NoticeTransactionError(txn, err)
		return utils.BadRequestResponse(c, "Invalid request payload")
	}

	nrpkg.AddTransactionAttribute(txn, "user.msisdn", beaconReq.MSISDN)
	nrpkg.AddTransactionAttribute(txn, "beacon.is_active", beaconReq.IsActive)

	err := h.userUC.UpdateBeaconStatus(c.Request().Context(), &beaconReq)
	if err != nil {
		nrpkg.NoticeTransactionError(txn, err)
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, "Failed to update beacon status")
	}

	return utils.SuccessResponse(c, http.StatusOK, "Beacon status updated successfully", nil)
}

package http

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/utils"
	"github.com/piresc/nebengjek/services/notification"
)

// NotificationHandler handles HTTP requests for notification operations
type NotificationHandler struct {
	notificationUC notification.NotificationUC
}

// NewNotificationHandler creates a new notification handler
func NewNotificationHandler(notificationUC notification.NotificationUC) *NotificationHandler {
	return &NotificationHandler{
		notificationUC: notificationUC,
	}
}

// GetNotificationHistory handles notification history retrieval requests
func (h *NotificationHandler) GetNotificationHistory(c echo.Context) error {
	userID := c.Param("user_id")
	if userID == "" {
		return utils.BadRequestResponse(c, "Invalid user ID")
	}

	// Parse query parameters
	limitStr := c.QueryParam("limit")
	offsetStr := c.QueryParam("offset")

	limit := 20 // default limit
	offset := 0 // default offset

	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	if offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	notifications, err := h.notificationUC.GetNotificationHistory(c.Request().Context(), userID, limit, offset)
	if err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, "Failed to retrieve notification history")
	}

	return utils.SuccessResponse(c, http.StatusOK, "Notification history retrieved successfully", notifications)
}

// GetUndeliveredNotifications handles undelivered notifications retrieval requests
func (h *NotificationHandler) GetUndeliveredNotifications(c echo.Context) error {
	userID := c.Param("user_id")
	if userID == "" {
		return utils.BadRequestResponse(c, "Invalid user ID")
	}

	notifications, err := h.notificationUC.GetUndeliveredNotifications(c.Request().Context(), userID)
	if err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, "Failed to retrieve undelivered notifications")
	}

	return utils.SuccessResponse(c, http.StatusOK, "Undelivered notifications retrieved successfully", notifications)
}

// MarkNotificationDelivered handles marking notifications as delivered
func (h *NotificationHandler) MarkNotificationDelivered(c echo.Context) error {
	notificationID := c.Param("notification_id")
	if notificationID == "" {
		return utils.BadRequestResponse(c, "Invalid notification ID")
	}

	err := h.notificationUC.MarkNotificationDelivered(c.Request().Context(), notificationID)
	if err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, "Failed to mark notification as delivered")
	}

	return utils.SuccessResponse(c, http.StatusOK, "Notification marked as delivered successfully", nil)
}
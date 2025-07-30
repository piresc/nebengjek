package handler

import (
	"github.com/labstack/echo/v4"
	httpHandler "github.com/piresc/nebengjek/services/notification/handler/http"
)

// RegisterRoutes registers all notification service routes
func RegisterRoutes(e *echo.Echo, notificationHandler *httpHandler.NotificationHandler) {
	// Note: Middleware is applied in main.go, not here
	
	// Notification routes
	api := e.Group("/api/v1")
	
	// Get notification history for a user
	api.GET("/notifications/history/:user_id", notificationHandler.GetNotificationHistory)
	
	// Get undelivered notifications for a user
	api.GET("/notifications/undelivered/:user_id", notificationHandler.GetUndeliveredNotifications)
	
	// Mark notification as delivered
	api.PUT("/notifications/:notification_id/delivered", notificationHandler.MarkNotificationDelivered)
}
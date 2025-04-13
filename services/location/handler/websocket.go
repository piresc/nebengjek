package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

// LocationUpdate represents a location update from a client
type LocationUpdate struct {
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Timestamp time.Time `json:"timestamp"`
}

// GeoLocation represents a geographic location
type GeoLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// WebSocketClient represents a connected WebSocket client
type WebSocketClient struct {
	UserID   string
	UserType string // "driver" or "customer"
	TripID   string
	Conn     *websocket.Conn
}

// WebSocketHandler handles WebSocket connections for real-time location updates
type WebSocketHandler struct {
	locationUsecase LocationUsecase
	locationService *LocationService
	natsProducer    NATSProducer
	clients         map[string]*WebSocketClient
}

// NATSProducer interface for publishing messages to NATS
type NATSProducer interface {
	Publish(topic string, message interface{}) error
}

// LocationUsecase interface for location operations
type LocationUsecase interface {
	UpdateLocation(userID, userType string, location GeoLocation) error
}

// LocationService manages WebSocket clients
type LocationService struct {
	clients map[string]*WebSocketClient
}

// NewLocationService creates a new location service
func NewLocationService() *LocationService {
	return &LocationService{
		clients: make(map[string]*WebSocketClient),
	}
}

// RegisterClient registers a new WebSocket client
func (s *LocationService) RegisterClient(client *WebSocketClient) {
	s.clients[client.UserID] = client
	log.Printf("Client registered: %s (%s)", client.UserID, client.UserType)
}

// UnregisterClient removes a WebSocket client
func (s *LocationService) UnregisterClient(client *WebSocketClient) {
	delete(s.clients, client.UserID)
	log.Printf("Client unregistered: %s", client.UserID)
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(locationUsecase LocationUsecase, natsProducer NATSProducer) *WebSocketHandler {
	return &WebSocketHandler{
		locationUsecase: locationUsecase,
		locationService: NewLocationService(),
		natsProducer:    natsProducer,
		clients:         make(map[string]*WebSocketClient),
	}
}

// HandleWebSocket handles WebSocket connections for location updates
func (h *WebSocketHandler) HandleWebSocket(c echo.Context) error {
	userID := c.Get("user_id").(string)     // From JWT token
	userType := c.Get("user_type").(string) // "driver" or "customer"
	tripID := c.QueryParam("trip_id")

	// Upgrade HTTP connection to WebSocket
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins in development
		},
	}

	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer ws.Close()

	// Register client
	client := &WebSocketClient{
		UserID:   userID,
		UserType: userType,
		TripID:   tripID,
		Conn:     ws,
	}

	h.locationService.RegisterClient(client)
	defer h.locationService.UnregisterClient(client)

	// Process incoming messages
	for {
		_, msg, err := ws.ReadMessage()
		if err != nil {
			break // Client disconnected
		}

		// Parse location update
		var locationUpdate LocationUpdate
		if err := json.Unmarshal(msg, &locationUpdate); err != nil {
			log.Printf("Error parsing location update: %v", err)
			continue
		}

		// Store location in Redis
		location := GeoLocation{
			Latitude:  locationUpdate.Latitude,
			Longitude: locationUpdate.Longitude,
		}

		err = h.locationUsecase.UpdateLocation(userID, userType, location)
		if err != nil {
			log.Printf("Error updating location: %v", err)
			continue
		}

		// If part of active trip, publish location update event
		if tripID != "" {
			locationEvent := LocationUpdateEvent{
				TripID:    tripID,
				UserID:    userID,
				UserType:  userType,
				Location:  location,
				Timestamp: time.Now(),
			}

			err = h.natsProducer.Publish("location.update", locationEvent)
			if err != nil {
				log.Printf("Error publishing location update: %v", err)
			}
		}
	}

	return nil
}

// LocationUpdateEvent represents a location update event for NATS
type LocationUpdateEvent struct {
	TripID    string      `json:"trip_id"`
	UserID    string      `json:"user_id"`
	UserType  string      `json:"user_type"`
	Location  GeoLocation `json:"location"`
	Timestamp time.Time   `json:"timestamp"`
}

package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock Match Repository
type MockMatchRepo struct {
	mock.Mock
}

func (m *MockMatchRepo) CreateMatch(ctx context.Context, match *models.Match) (*models.Match, error) {
	args := m.Called(ctx, match)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Match), args.Error(1)
}

func (m *MockMatchRepo) GetMatch(ctx context.Context, matchID string) (*models.Match, error) {
	args := m.Called(ctx, matchID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Match), args.Error(1)
}

func (m *MockMatchRepo) UpdateMatchStatus(ctx context.Context, id string, status models.MatchStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockMatchRepo) ListMatchesByDriver(ctx context.Context, driverID string) ([]*models.Match, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Match), args.Error(1)
}

func (m *MockMatchRepo) ListMatchesByPassenger(ctx context.Context, passengerID uuid.UUID) ([]*models.Match, error) {
	args := m.Called(ctx, passengerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Match), args.Error(1)
}

func (m *MockMatchRepo) StoreMatchProposal(ctx context.Context, match *models.Match) error {
	args := m.Called(ctx, match)
	return args.Error(0)
}

func (m *MockMatchRepo) ConfirmMatchAtomically(ctx context.Context, matchID string, status models.MatchStatus) error {
	args := m.Called(ctx, matchID, status)
	return args.Error(0)
}

func (m *MockMatchRepo) AddAvailableDriver(ctx context.Context, driverID string, location *models.Location) error {
	args := m.Called(ctx, driverID, location)
	return args.Error(0)
}

func (m *MockMatchRepo) AddAvailablePassenger(ctx context.Context, passengerID string, location *models.Location) error {
	args := m.Called(ctx, passengerID, location)
	return args.Error(0)
}

func (m *MockMatchRepo) RemoveAvailableDriver(ctx context.Context, driverID string) error {
	args := m.Called(ctx, driverID)
	return args.Error(0)
}

func (m *MockMatchRepo) RemoveAvailablePassenger(ctx context.Context, passengerID string) error {
	args := m.Called(ctx, passengerID)
	return args.Error(0)
}

func (m *MockMatchRepo) FindNearbyDrivers(ctx context.Context, location *models.Location, radius float64) ([]models.NearbyUser, error) {
	args := m.Called(ctx, location, radius)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.NearbyUser), args.Error(1)
}

func (m *MockMatchRepo) FindNearbyPassengers(ctx context.Context, location *models.Location, radius float64) ([]models.NearbyUser, error) {
	args := m.Called(ctx, location, radius)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.NearbyUser), args.Error(1)
}

// Mock Match Gateway
type MockMatchGW struct {
	mock.Mock
}

func (m *MockMatchGW) PublishMatchFound(ctx context.Context, mp models.MatchProposal) error {
	args := m.Called(ctx, mp)
	return args.Error(0)
}

func (m *MockMatchGW) PublishMatchConfirm(ctx context.Context, mp models.MatchProposal) error {
	args := m.Called(ctx, mp)
	return args.Error(0)
}

func (m *MockMatchGW) PublishMatchRejected(ctx context.Context, mp models.MatchProposal) error {
	args := m.Called(ctx, mp)
	return args.Error(0)
}

func TestHandleBeaconEvent_ActiveDriver(t *testing.T) {
	// Arrange
	mockRepo := new(MockMatchRepo)
	mockGW := new(MockMatchGW)

	uc := NewMatchUC(mockRepo, mockGW)

	driverEvent := models.BeaconEvent{
		UserID:   uuid.New().String(),
		Role:     "driver",
		IsActive: true,
		Location: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
		},
	}

	mockRepo.On("AddAvailableDriver", mock.Anything, driverEvent.UserID, mock.MatchedBy(func(loc *models.Location) bool {
		return loc.Latitude == driverEvent.Location.Latitude && loc.Longitude == driverEvent.Location.Longitude
	})).Return(nil)

	// Act
	err := uc.HandleBeaconEvent(driverEvent)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestHandleBeaconEvent_ActivePassenger(t *testing.T) {
	// Arrange
	mockRepo := new(MockMatchRepo)
	mockGW := new(MockMatchGW)

	uc := NewMatchUC(mockRepo, mockGW)

	passengerEvent := models.BeaconEvent{
		UserID:   uuid.New().String(),
		Role:     "passenger",
		IsActive: true,
		Location: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
		},
	}

	// Mock adding passenger
	mockRepo.On("AddAvailablePassenger", mock.Anything, passengerEvent.UserID, mock.MatchedBy(func(loc *models.Location) bool {
		return loc.Latitude == passengerEvent.Location.Latitude && loc.Longitude == passengerEvent.Location.Longitude
	})).Return(nil)

	// Mock finding nearby drivers
	nearbyDrivers := []models.NearbyUser{
		{
			ID: "driver-123",
			Location: &models.Location{
				Latitude:  -6.175492,
				Longitude: 106.827253,
			},
			Distance: 0.2, // km
		},
	}

	mockRepo.On("FindNearbyDrivers", mock.Anything, mock.MatchedBy(func(loc *models.Location) bool {
		return loc.Latitude == passengerEvent.Location.Latitude && loc.Longitude == passengerEvent.Location.Longitude
	}), mock.Anything).Return(nearbyDrivers, nil)

	// Mock creating match
	mockMatch := &models.Match{
		ID:          uuid.New(),
		DriverID:    uuid.MustParse("00000000-0000-0000-0000-000000000000"), // Will be overwritten
		PassengerID: uuid.MustParse(passengerEvent.UserID),
		Status:      models.MatchStatusPending,
	}

	mockRepo.On("CreateMatch", mock.Anything, mock.MatchedBy(func(m *models.Match) bool {
		m.ID = mockMatch.ID // Set the ID for the created match
		return true
	})).Return(mockMatch, nil)

	// Mock storing match proposal
	mockRepo.On("StoreMatchProposal", mock.Anything, mockMatch).Return(nil)

	// Mock publishing match proposal
	mockGW.On("PublishMatchFound", mock.Anything, mock.MatchedBy(func(mp models.MatchProposal) bool {
		return true // Simplified match for easier testing
	})).Return(nil)

	// Act
	err := uc.HandleBeaconEvent(passengerEvent)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockGW.AssertExpectations(t)
}

func TestHandleBeaconEvent_InactiveDriver(t *testing.T) {
	// Arrange
	mockRepo := new(MockMatchRepo)
	mockGW := new(MockMatchGW)

	uc := NewMatchUC(mockRepo, mockGW)

	driverEvent := models.BeaconEvent{
		UserID:   uuid.New().String(),
		Role:     "driver",
		IsActive: false,
	}

	mockRepo.On("RemoveAvailableDriver", mock.Anything, driverEvent.UserID).Return(nil)

	// Act
	err := uc.HandleBeaconEvent(driverEvent)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestHandleBeaconEvent_InactivePassenger(t *testing.T) {
	// Arrange
	mockRepo := new(MockMatchRepo)
	mockGW := new(MockMatchGW)

	uc := NewMatchUC(mockRepo, mockGW)

	passengerEvent := models.BeaconEvent{
		UserID:   uuid.New().String(),
		Role:     "passenger",
		IsActive: false,
	}

	mockRepo.On("RemoveAvailablePassenger", mock.Anything, passengerEvent.UserID).Return(nil)

	// Act
	err := uc.HandleBeaconEvent(passengerEvent)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestHandleBeaconEvent_RepositoryError(t *testing.T) {
	// Arrange
	mockRepo := new(MockMatchRepo)
	mockGW := new(MockMatchGW)

	uc := NewMatchUC(mockRepo, mockGW)

	driverEvent := models.BeaconEvent{
		UserID:   uuid.New().String(),
		Role:     "driver",
		IsActive: true,
		Location: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
		},
	}

	expectedError := errors.New("repository error")
	mockRepo.On("AddAvailableDriver", mock.Anything, driverEvent.UserID, mock.Anything).Return(expectedError)

	// Act
	err := uc.HandleBeaconEvent(driverEvent)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	mockRepo.AssertExpectations(t)
}

func TestHandleBeaconEvent_NoNearbyDrivers(t *testing.T) {
	// Arrange
	mockRepo := new(MockMatchRepo)
	mockGW := new(MockMatchGW)

	uc := NewMatchUC(mockRepo, mockGW)

	passengerEvent := models.BeaconEvent{
		UserID:   uuid.New().String(),
		Role:     "passenger",
		IsActive: true,
		Location: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
		},
	}

	// Mock adding passenger
	mockRepo.On("AddAvailablePassenger", mock.Anything, passengerEvent.UserID, mock.MatchedBy(func(loc *models.Location) bool {
		return loc.Latitude == passengerEvent.Location.Latitude && loc.Longitude == passengerEvent.Location.Longitude
	})).Return(nil)

	// Mock finding nearby drivers - empty result
	var emptyNearbyUsers []models.NearbyUser
	mockRepo.On("FindNearbyDrivers", mock.Anything, mock.Anything, mock.Anything).Return(emptyNearbyUsers, nil)

	// Act
	err := uc.HandleBeaconEvent(passengerEvent)

	// Assert
	assert.NoError(t, err) // Should still succeed even without drivers
	mockRepo.AssertExpectations(t)
}

func TestConfirmMatchStatus_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockMatchRepo)
	mockGW := new(MockMatchGW)

	uc := NewMatchUC(mockRepo, mockGW)

	matchID := uuid.New().String()
	mp := models.MatchProposal{
		ID:          matchID,
		DriverID:    "driver-123",
		PassengerID: "passenger-456",
		MatchStatus: models.MatchStatusAccepted,
	}

	match := &models.Match{
		ID:          uuid.MustParse(matchID),
		DriverID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		PassengerID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		Status:      models.MatchStatusPending,
	}

	mockRepo.On("GetMatch", mock.Anything, matchID).Return(match, nil)
	mockRepo.On("UpdateMatchStatus", mock.Anything, matchID, models.MatchStatusAccepted).Return(nil)
	mockGW.On("PublishMatchConfirm", mock.Anything, mock.MatchedBy(func(proposal models.MatchProposal) bool {
		return proposal.ID == matchID && proposal.MatchStatus == models.MatchStatusAccepted
	})).Return(nil)

	// Act
	err := uc.ConfirmMatchStatus(matchID, mp)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockGW.AssertExpectations(t)
}

func TestConfirmMatchStatus_InvalidMatch(t *testing.T) {
	// Arrange
	mockRepo := new(MockMatchRepo)
	mockGW := new(MockMatchGW)

	uc := NewMatchUC(mockRepo, mockGW)

	matchID := uuid.New().String()
	mp := models.MatchProposal{
		ID:          matchID,
		DriverID:    "driver-123",
		PassengerID: "passenger-456",
		MatchStatus: models.MatchStatusAccepted,
	}

	mockRepo.On("GetMatch", mock.Anything, matchID).Return(nil, errors.New("match not found"))

	// Act
	err := uc.ConfirmMatchStatus(matchID, mp)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "match not found")
	mockRepo.AssertExpectations(t)
}

func TestConfirmMatchStatus_MismatchedDriverPassenger(t *testing.T) {
	// Arrange
	mockRepo := new(MockMatchRepo)
	mockGW := new(MockMatchGW)

	uc := NewMatchUC(mockRepo, mockGW)

	matchID := uuid.New().String()
	mp := models.MatchProposal{
		ID:          matchID,
		DriverID:    "driver-123",
		PassengerID: "passenger-456",
		MatchStatus: models.MatchStatusAccepted,
	}

	// Different driverID than what's in the proposal
	match := &models.Match{
		ID:          uuid.MustParse(matchID),
		DriverID:    uuid.MustParse("00000000-0000-0000-0000-000000000009"), // Different from mp.DriverID
		PassengerID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		Status:      models.MatchStatusPending,
	}

	mockRepo.On("GetMatch", mock.Anything, matchID).Return(match, nil)

	// Act
	err := uc.ConfirmMatchStatus(matchID, mp)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mismatch in match data")
	mockRepo.AssertExpectations(t)
}

func TestConfirmMatchStatus_UpdateError(t *testing.T) {
	// Arrange
	mockRepo := new(MockMatchRepo)
	mockGW := new(MockMatchGW)

	uc := NewMatchUC(mockRepo, mockGW)

	matchID := uuid.New().String()
	mp := models.MatchProposal{
		ID:          matchID,
		DriverID:    "driver-123",
		PassengerID: "passenger-456",
		MatchStatus: models.MatchStatusAccepted,
	}

	match := &models.Match{
		ID:          uuid.MustParse(matchID),
		DriverID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		PassengerID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		Status:      models.MatchStatusPending,
	}

	updateError := errors.New("update error")
	mockRepo.On("GetMatch", mock.Anything, matchID).Return(match, nil)
	mockRepo.On("UpdateMatchStatus", mock.Anything, matchID, models.MatchStatusAccepted).Return(updateError)

	// Act
	err := uc.ConfirmMatchStatus(matchID, mp)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, updateError, err)
	mockRepo.AssertExpectations(t)
}

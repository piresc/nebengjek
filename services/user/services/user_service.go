package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/user/repository"
	"golang.org/x/crypto/bcrypt"
)

// UserService defines the interface for user business logic operations
type UserService interface {
	// User operations
	RegisterUser(ctx context.Context, user *models.User) error
	AuthenticateUser(ctx context.Context, email, password string) (*models.User, error)
	GetUserByID(ctx context.Context, id string) (*models.User, error)
	UpdateUserProfile(ctx context.Context, user *models.User) error
	DeactivateUser(ctx context.Context, id string) error
	ListUsers(ctx context.Context, offset, limit int) ([]*models.User, error)

	// Driver-specific operations
	RegisterDriver(ctx context.Context, user *models.User) error
	UpdateDriverLocation(ctx context.Context, driverID string, location *models.Location) error
	UpdateDriverAvailability(ctx context.Context, driverID string, isAvailable bool) error
	GetNearbyDrivers(ctx context.Context, location *models.Location, radiusKm float64) ([]*models.User, error)
	VerifyDriver(ctx context.Context, driverID string) error
}

// userService implements UserService interface
type userService struct {
	repo repository.UserRepository
}

// NewUserService creates a new user service
func NewUserService(repo repository.UserRepository) UserService {
	return &userService{repo: repo}
}

// RegisterUser registers a new user
func (s *userService) RegisterUser(ctx context.Context, user *models.User) error {
	// Validate user data
	if err := validateUserData(user); err != nil {
		return err
	}

	// Generate UUID if not provided
	if user.ID == "" {
		user.ID = uuid.New().String()
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	user.Password = string(hashedPassword)

	// Set default values
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	user.IsActive = true
	user.Rating = 0

	// Set role to passenger if not specified
	if user.Role == "" {
		user.Role = "passenger"
	}

	// Create user
	return s.repo.CreateUser(ctx, user)
}

// AuthenticateUser authenticates a user by email and password
func (s *userService) AuthenticateUser(ctx context.Context, email, password string) (*models.User, error) {
	// Get user by email
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	// Check if user is active
	if !user.IsActive {
		return nil, fmt.Errorf("user account is deactivated")
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	// Clear password before returning
	user.Password = ""

	return user, nil
}

// GetUserByID retrieves a user by ID
func (s *userService) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	user, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Clear password before returning
	user.Password = ""

	return user, nil
}

// UpdateUserProfile updates a user's profile
func (s *userService) UpdateUserProfile(ctx context.Context, user *models.User) error {
	// Validate user data
	if err := validateUserData(user); err != nil {
		return err
	}

	// Get existing user
	existingUser, err := s.repo.GetUserByID(ctx, user.ID)
	if err != nil {
		return err
	}

	// Update only allowed fields
	existingUser.Email = user.Email
	existingUser.PhoneNumber = user.PhoneNumber
	existingUser.FullName = user.FullName
	existingUser.UpdatedAt = time.Now()

	// Update password if provided
	if user.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("failed to hash password: %w", err)
		}
		existingUser.Password = string(hashedPassword)
	}

	// Update driver info if user is a driver
	if existingUser.Role == "driver" && user.DriverInfo != nil {
		existingUser.DriverInfo.VehicleType = user.DriverInfo.VehicleType
		existingUser.DriverInfo.VehiclePlate = user.DriverInfo.VehiclePlate
		existingUser.DriverInfo.VehicleModel = user.DriverInfo.VehicleModel
		existingUser.DriverInfo.VehicleColor = user.DriverInfo.VehicleColor
		existingUser.DriverInfo.LicenseNumber = user.DriverInfo.LicenseNumber
		existingUser.DriverInfo.Documents = user.DriverInfo.Documents
	}

	// Update user
	return s.repo.UpdateUser(ctx, existingUser)
}

// DeactivateUser deactivates a user account
func (s *userService) DeactivateUser(ctx context.Context, id string) error {
	// Get existing user
	user, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		return err
	}

	// Deactivate user
	user.IsActive = false
	user.UpdatedAt = time.Now()

	// Update user
	return s.repo.UpdateUser(ctx, user)
}

// ListUsers retrieves a list of users with pagination
func (s *userService) ListUsers(ctx context.Context, offset, limit int) ([]*models.User, error) {
	users, err := s.repo.ListUsers(ctx, offset, limit)
	if err != nil {
		return nil, err
	}

	// Clear passwords before returning
	for _, user := range users {
		user.Password = ""
	}

	return users, nil
}

// RegisterDriver registers a new driver
func (s *userService) RegisterDriver(ctx context.Context, user *models.User) error {
	// Validate user data
	if err := validateUserData(user); err != nil {
		return err
	}

	// Validate driver data
	if err := validateDriverData(user.DriverInfo); err != nil {
		return err
	}

	// Set role to driver
	user.Role = "driver"

	// Set default driver values
	user.DriverInfo.Verified = false
	user.DriverInfo.IsAvailable = false

	// Register user
	return s.RegisterUser(ctx, user)
}

// UpdateDriverLocation updates a driver's current location
func (s *userService) UpdateDriverLocation(ctx context.Context, driverID string, location *models.Location) error {
	// Validate location data
	if err := validateLocationData(location); err != nil {
		return err
	}

	// Get existing user
	user, err := s.repo.GetUserByID(ctx, driverID)
	if err != nil {
		return err
	}

	// Check if user is a driver
	if user.Role != "driver" {
		return fmt.Errorf("user is not a driver")
	}

	// Set timestamp if not provided
	if location.Timestamp.IsZero() {
		location.Timestamp = time.Now()
	}

	// Update driver location
	return s.repo.UpdateDriverLocation(ctx, driverID, location)
}

// UpdateDriverAvailability updates a driver's availability status
func (s *userService) UpdateDriverAvailability(ctx context.Context, driverID string, isAvailable bool) error {
	// Get existing user
	user, err := s.repo.GetUserByID(ctx, driverID)
	if err != nil {
		return err
	}

	// Check if user is a driver
	if user.Role != "driver" {
		return fmt.Errorf("user is not a driver")
	}

	// Check if driver is verified
	if !user.DriverInfo.Verified {
		return fmt.Errorf("driver is not verified")
	}

	// Update driver availability
	return s.repo.UpdateDriverAvailability(ctx, driverID, isAvailable)
}

// GetNearbyDrivers retrieves available drivers near a location within a radius
func (s *userService) GetNearbyDrivers(ctx context.Context, location *models.Location, radiusKm float64) ([]*models.User, error) {
	// Validate location data
	if err := validateLocationData(location); err != nil {
		return nil, err
	}

	// Validate radius
	if radiusKm <= 0 {
		return nil, fmt.Errorf("radius must be positive")
	}

	// Get nearby drivers
	drivers, err := s.repo.GetNearbyDrivers(ctx, location, radiusKm)
	if err != nil {
		return nil, err
	}

	// Clear passwords before returning
	for _, driver := range drivers {
		driver.Password = ""
	}

	return drivers, nil
}

// VerifyDriver marks a driver as verified
func (s *userService) VerifyDriver(ctx context.Context, driverID string) error {
	// Get existing user
	user, err := s.repo.GetUserByID(ctx, driverID)
	if err != nil {
		return err
	}

	// Check if user is a driver
	if user.Role != "driver" {
		return fmt.Errorf("user is not a driver")
	}

	// Check if driver is already verified
	if user.DriverInfo.Verified {
		return fmt.Errorf("driver is already verified")
	}

	// Verify driver
	return s.repo.VerifyDriver(ctx, driverID)
}

// Helper functions for validation

func validateUserData(user *models.User) error {
	if user == nil {
		return errors.New("user cannot be nil")
	}

	if user.Email == "" {
		return errors.New("email is required")
	}

	if user.PhoneNumber == "" {
		return errors.New("phone number is required")
	}

	if user.FullName == "" {
		return errors.New("full name is required")
	}

	return nil
}

func validateDriverData(driver *models.Driver) error {
	if driver == nil {
		return errors.New("driver info cannot be nil")
	}

	if driver.VehicleType == "" {
		return errors.New("vehicle type is required")
	}

	if driver.VehiclePlate == "" {
		return errors.New("vehicle plate is required")
	}

	if driver.VehicleModel == "" {
		return errors.New("vehicle model is required")
	}

	if driver.VehicleColor == "" {
		return errors.New("vehicle color is required")
	}

	if driver.LicenseNumber == "" {
		return errors.New("license number is required")
	}

	return nil
}

func validateLocationData(location *models.Location) error {
	if location == nil {
		return errors.New("location cannot be nil")
	}

	if location.Latitude < -90 || location.Latitude > 90 {
		return errors.New("latitude must be between -90 and 90")
	}

	if location.Longitude < -180 || location.Longitude > 180 {
		return errors.New("longitude must be between -180 and 180")
	}

	return nil
}

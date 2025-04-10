package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/internal/utils"
	"github.com/piresc/nebengjek/services/user"
)

// UserUC implements UserUC interface
type UserUC struct {
	repo user.UserRepo
}

// NewUserUC creates a new user UC
func NewUserUC(repo user.UserRepo) *UserUC {
	return &UserUC{repo: repo}
}

// RegisterUser registers a new user
func (s *UserUC) RegisterUser(ctx context.Context, user *models.User) error {
	// Validate user data
	if err := validateUserData(user); err != nil {
		return err
	}

	// Validate MSISDN format
	isValid, formattedMSISDN, err := utils.ValidateMSISDN(user.MSISDN)
	if err != nil || !isValid {
		return fmt.Errorf("invalid MSISDN format or not a Telkomsel number")
	}
	user.MSISDN = formattedMSISDN

	// Generate UUID if not provided
	if user.ID == "" {
		user.ID = uuid.New().String()
	}

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

// GetUserByID retrieves a user by ID
func (s *UserUC) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	user, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// UpdateUserProfile updates a user's profile
func (s *UserUC) UpdateUserProfile(ctx context.Context, user *models.User) error {
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
	existingUser.FullName = user.FullName
	existingUser.UpdatedAt = time.Now()

	// Update MSISDN if provided and valid
	if user.MSISDN != "" && user.MSISDN != existingUser.MSISDN {
		isValid, formattedMSISDN, err := utils.ValidateMSISDN(user.MSISDN)
		if err != nil || !isValid {
			return fmt.Errorf("invalid MSISDN format or not a Telkomsel number")
		}
		existingUser.MSISDN = formattedMSISDN
	}

	// Update driver info if user is a driver
	if existingUser.Role == "driver" && user.DriverInfo != nil {
		existingUser.DriverInfo.VehicleType = user.DriverInfo.VehicleType
		existingUser.DriverInfo.VehiclePlate = user.DriverInfo.VehiclePlate
		existingUser.DriverInfo.VehicleModel = user.DriverInfo.VehicleModel
		existingUser.DriverInfo.VehicleColor = user.DriverInfo.VehicleColor
		existingUser.DriverInfo.LicenseNumber = user.DriverInfo.LicenseNumber
	}

	// Update user
	return s.repo.UpdateUser(ctx, existingUser)
}

// DeactivateUser deactivates a user account
func (s *UserUC) DeactivateUser(ctx context.Context, id string) error {
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
func (s *UserUC) ListUsers(ctx context.Context, offset, limit int) ([]*models.User, error) {
	users, err := s.repo.ListUsers(ctx, offset, limit)
	if err != nil {
		return nil, err
	}

	return users, nil
}

// RegisterDriver registers a new driver
func (s *UserUC) RegisterDriver(ctx context.Context, user *models.User) error {
	// Validate user data
	if err := validateUserData(user); err != nil {
		return err
	}

	// Validate MSISDN format
	isValid, formattedMSISDN, err := utils.ValidateMSISDN(user.MSISDN)
	if err != nil || !isValid {
		return fmt.Errorf("invalid MSISDN format or not a Telkomsel number")
	}
	user.MSISDN = formattedMSISDN

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

// Helper functions for validation

func validateUserData(user *models.User) error {
	if user == nil {
		return errors.New("user cannot be nil")
	}

	if user.MSISDN == "" {
		return errors.New("MSISDN is required")
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

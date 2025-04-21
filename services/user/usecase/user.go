package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/internal/utils"
)

// RegisterUser registers a new user
func (u *UserUC) RegisterUser(ctx context.Context, user *models.User) error {
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
	user.IsActive = true

	// Set role to passenger if not specified
	if user.Role == "" {
		user.Role = "passenger"
	}

	// Create user
	return u.userRepo.CreateUser(ctx, user)
}

// GetUserByID retrieves a user by ID
func (u *UserUC) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	user, err := u.userRepo.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// RegisterDriver registers a new driver
func (u *UserUC) RegisterDriver(ctx context.Context, userDriver *models.User) error {

	// Validate MSISDN format
	isValid, formattedMSISDN, err := utils.ValidateMSISDN(userDriver.MSISDN)
	if err != nil || !isValid {
		return fmt.Errorf("invalid MSISDN format or not a Telkomsel number")
	}
	userDriver.MSISDN = formattedMSISDN

	user, err := u.userRepo.GetUserByMSISDN(ctx, userDriver.MSISDN)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	if user.Role == "driver" {
		return fmt.Errorf("user is already registered as a driver")
	}
	// Validate user data
	if err := validateUserData(userDriver); err != nil {
		return err
	}

	// Validate driver data
	if err := validateDriverData(userDriver.DriverInfo); err != nil {
		return err
	}

	// Set role to driver
	userDriver.Role = "driver"
	userDriver.ID = user.ID

	// Register user
	return u.userRepo.UpdateToDriver(ctx, userDriver)
}

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
	return nil
}

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
	// Validate user data first
	if err := validateUserData(user); err != nil {
		return err
	}

	// Validate MSISDN format
	isValid, formattedMSISDN, err := utils.ValidateMSISDN(user.MSISDN)
	if err != nil || !isValid {
		validationErr := fmt.Errorf("invalid MSISDN format or not a Telkomsel number")
		return validationErr
	}
	user.MSISDN = formattedMSISDN
	user.IsActive = true

	// Set role to passenger if not specified
	if user.Role == "" {
		user.Role = "passenger"
	}

	// Create user
	err = u.userRepo.CreateUser(ctx, user)
	if err != nil {
		return err
	}

	return nil
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
		validationErr := fmt.Errorf("invalid MSISDN format or not a Telkomsel number")
		return validationErr
	}
	userDriver.MSISDN = formattedMSISDN

	user, err := u.userRepo.GetUserByMSISDN(ctx, userDriver.MSISDN)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	if user.Role == "driver" {
		existingDriverErr := fmt.Errorf("user is already registered as a driver")
		return existingDriverErr
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
	err = u.userRepo.UpdateToDriver(ctx, userDriver)
	if err != nil {
		return err
	}

	return nil
}

func validateUserData(user *models.User) error {
	if user == nil {
		return errors.New("user cannot be nil")
	}

	if user.MSISDN == "" {
		return errors.New("MSISDN is required")
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

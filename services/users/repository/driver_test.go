package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

func TestGetDriverInfo(t *testing.T) {
	testCases := []struct {
		name       string
		userID     uuid.UUID
		mockSetup  func(mock sqlmock.Sqlmock)
		assertFunc func(t *testing.T, driver *models.Driver, err error)
	}{
		{
			name:   "Success",
			userID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
			mockSetup: func(mock sqlmock.Sqlmock) {
				userID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")
				rows := sqlmock.NewRows([]string{"user_id", "vehicle_type", "vehicle_plate"}).
					AddRow(userID, "car", "B 1234 ABC")
				mock.ExpectQuery("^SELECT \\* FROM drivers WHERE user_id").
					WithArgs(userID).
					WillReturnRows(rows)
			},
			assertFunc: func(t *testing.T, driver *models.Driver, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, driver)
				assert.Equal(t, "car", driver.VehicleType)
				assert.Equal(t, "B 1234 ABC", driver.VehiclePlate)
			},
		},
		{
			name:   "Driver Not Found",
			userID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440002"),
			mockSetup: func(mock sqlmock.Sqlmock) {
				userID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440002")
				mock.ExpectQuery("^SELECT \\* FROM drivers WHERE user_id").
					WithArgs(userID).
					WillReturnError(sql.ErrNoRows)
			},
			assertFunc: func(t *testing.T, driver *models.Driver, err error) {
				assert.NoError(t, err)
				assert.Nil(t, driver)
			},
		},
		{
			name:   "Database Error",
			userID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440003"),
			mockSetup: func(mock sqlmock.Sqlmock) {
				userID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440003")
				mock.ExpectQuery("^SELECT \\* FROM drivers WHERE user_id").
					WithArgs(userID).
					WillReturnError(errors.New("database error"))
			},
			assertFunc: func(t *testing.T, driver *models.Driver, err error) {
				assert.Error(t, err)
				assert.Nil(t, driver)
				assert.Contains(t, err.Error(), "failed to get driver info")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			repo, mock, cleanup := setupUserRepoTest(t)
			defer cleanup()

			// Apply mocks
			tc.mockSetup(mock)

			// Execute
			driver, err := repo.getDriverInfo(context.Background(), tc.userID)

			// Assert
			tc.assertFunc(t, driver, err)

			// Verify all expectations were met
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUpdateToDriver(t *testing.T) {
	testCases := []struct {
		name       string
		user       models.User
		mockSetup  func(mock sqlmock.Sqlmock)
		assertFunc func(t *testing.T, err error)
	}{
		{
			name: "Success",
			user: models.User{
				ID:       uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
				MSISDN:   "+628123456789",
				FullName: "John Doe",
				Role:     "driver",
				IsActive: true,
				DriverInfo: &models.Driver{
					VehicleType:  "car",
					VehiclePlate: "B 1234 ABC",
				},
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("^UPDATE users").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec("^INSERT INTO drivers").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			assertFunc: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "Begin Transaction Error",
			user: models.User{
				ID:       uuid.MustParse("550e8400-e29b-41d4-a716-446655440002"),
				MSISDN:   "+628123456790",
				FullName: "Jane Doe",
				Role:     "driver",
				DriverInfo: &models.Driver{
					VehicleType:  "motorcycle",
					VehiclePlate: "B 5678 DEF",
				},
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin().WillReturnError(errors.New("begin transaction error"))
			},
			assertFunc: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to begin transaction")
			},
		},
		{
			name: "Update User Error",
			user: models.User{
				ID:       uuid.MustParse("550e8400-e29b-41d4-a716-446655440003"),
				MSISDN:   "+628123456791",
				FullName: "Error User",
				Role:     "driver",
				DriverInfo: &models.Driver{
					VehicleType:  "car",
					VehiclePlate: "B 9012 GHI",
				},
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("^UPDATE users").
					WillReturnError(errors.New("update user error"))
				mock.ExpectRollback()
			},
			assertFunc: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to update user role")
			},
		},
		{
			name: "Insert Driver Info Error",
			user: models.User{
				ID:       uuid.MustParse("550e8400-e29b-41d4-a716-446655440004"),
				MSISDN:   "+628123456792",
				FullName: "Driver Info Error",
				Role:     "driver",
				DriverInfo: &models.Driver{
					VehicleType:  "car",
					VehiclePlate: "B 3456 JKL",
				},
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("^UPDATE users").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec("^INSERT INTO drivers").
					WillReturnError(errors.New("insert driver info error"))
				mock.ExpectRollback()
			},
			assertFunc: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to insert driver info")
			},
		},
		{
			name: "Commit Error",
			user: models.User{
				ID:       uuid.MustParse("550e8400-e29b-41d4-a716-446655440005"),
				MSISDN:   "+628123456793",
				FullName: "Commit Error",
				Role:     "driver",
				DriverInfo: &models.Driver{
					VehicleType:  "car",
					VehiclePlate: "B 7890 MNO",
				},
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("^UPDATE users").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec("^INSERT INTO drivers").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit().WillReturnError(errors.New("commit error"))
			},
			assertFunc: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to commit transaction")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			repo, mock, cleanup := setupUserRepoTest(t)
			defer cleanup()

			// Apply mocks
			tc.mockSetup(mock)

			// Execute
			err := repo.UpdateToDriver(context.Background(), &tc.user)

			// Assert
			tc.assertFunc(t, err)

			// Verify all expectations were met
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestGetUserByID(t *testing.T) {
	testCases := []struct {
		name       string
		userID     string
		mockSetup  func(mock sqlmock.Sqlmock)
		assertFunc func(t *testing.T, user *models.User, err error)
	}{
		{
			name:   "Success",
			userID: "550e8400-e29b-41d4-a716-446655440001",
			mockSetup: func(mock sqlmock.Sqlmock) {
				userUUID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")
				rows := sqlmock.NewRows([]string{"id", "msisdn", "fullname", "role", "created_at", "updated_at", "is_active"}).
					AddRow(userUUID, "+628123456789", "John Doe", "user", time.Now(), time.Now(), true)
				mock.ExpectQuery("^SELECT \\* FROM users WHERE").
					WithArgs("550e8400-e29b-41d4-a716-446655440001").
					WillReturnRows(rows)
			},
			assertFunc: func(t *testing.T, user *models.User, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, "+628123456789", user.MSISDN)
				assert.Equal(t, "John Doe", user.FullName)
				assert.Equal(t, "user", user.Role)
			},
		},
		{
			name:   "User Not Found",
			userID: "550e8400-e29b-41d4-a716-446655440099",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("^SELECT \\* FROM users WHERE").
					WithArgs("550e8400-e29b-41d4-a716-446655440099").
					WillReturnError(sql.ErrNoRows)
			},
			assertFunc: func(t *testing.T, user *models.User, err error) {
				assert.Error(t, err)
				assert.Nil(t, user)
				assert.Contains(t, err.Error(), "user not found")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			repo, mock, cleanup := setupUserRepoTest(t)
			defer cleanup()

			// Apply mocks
			tc.mockSetup(mock)

			// Execute
			user, err := repo.GetUserByID(context.Background(), tc.userID)

			// Assert
			tc.assertFunc(t, user, err)

			// Verify all expectations were met
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
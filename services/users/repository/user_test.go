package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/piresc/nebengjek/internal/pkg/database"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

func setupUserRepoTest(t *testing.T) (*UserRepo, sqlmock.Sqlmock, func()) {
	// Create SQL mock
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	// Create sqlx DB with mock
	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	
	// Create a mock Redis client (nil for now as we're not testing Redis operations in user.go)
	redisClient := &database.RedisClient{}

	// Create repo with mocks
	repo := &UserRepo{
		db:          sqlxDB,
		redisClient: redisClient,
		cfg:         &models.Config{},
	}

	// Return cleanup function
	cleanup := func() {
		sqlxDB.Close()
	}

	return repo, mock, cleanup
}

func TestGetUserByMSISDN(t *testing.T) {
	// Test cases
	testCases := []struct {
		name       string
		msisdn     string
		mockSetup  func(mock sqlmock.Sqlmock)
		assertFunc func(t *testing.T, user *models.User, err error)
	}{
		{
			name:   "Success - Regular User",
			msisdn: "+628123456789",
			mockSetup: func(mock sqlmock.Sqlmock) {
				userID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
				rows := sqlmock.NewRows([]string{"id", "msisdn", "fullname", "role", "created_at", "updated_at", "is_active"}).
					AddRow(userID, "+628123456789", "John Doe", "user", time.Now(), time.Now(), true)
				mock.ExpectQuery("^SELECT (.+) FROM users WHERE msisdn").
					WithArgs("+628123456789").
					WillReturnRows(rows)
			},
			assertFunc: func(t *testing.T, user *models.User, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, "+628123456789", user.MSISDN)
				assert.Equal(t, "John Doe", user.FullName)
				assert.Equal(t, "user", user.Role)
				assert.True(t, user.IsActive)
				assert.Nil(t, user.DriverInfo)
			},
		},
		{
			name:   "Success - Driver User",
			msisdn: "+628123456790",
			mockSetup: func(mock sqlmock.Sqlmock) {
				userID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")
				rows := sqlmock.NewRows([]string{"id", "msisdn", "fullname", "role", "created_at", "updated_at", "is_active"}).
					AddRow(userID, "+628123456790", "Jane Driver", "driver", time.Now(), time.Now(), true)
				mock.ExpectQuery("^SELECT (.+) FROM users WHERE msisdn").
					WithArgs("+628123456790").
					WillReturnRows(rows)

				// Mock driver info query
				driverRows := sqlmock.NewRows([]string{"user_id", "vehicle_type", "vehicle_plate"}).
					AddRow(userID, "car", "B 1234 ABC")
				mock.ExpectQuery("^SELECT \\* FROM drivers WHERE user_id").
					WithArgs(userID).
					WillReturnRows(driverRows)
			},
			assertFunc: func(t *testing.T, user *models.User, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, "+628123456790", user.MSISDN)
				assert.Equal(t, "Jane Driver", user.FullName)
				assert.Equal(t, "driver", user.Role)
				assert.NotNil(t, user.DriverInfo)
				assert.Equal(t, "car", user.DriverInfo.VehicleType)
				assert.Equal(t, "B 1234 ABC", user.DriverInfo.VehiclePlate)
			},
		},
		{
			name:   "User Not Found",
			msisdn: "+628999999999",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("^SELECT (.+) FROM users WHERE msisdn").
					WithArgs("+628999999999").
					WillReturnError(sql.ErrNoRows)
			},
			assertFunc: func(t *testing.T, user *models.User, err error) {
				assert.Error(t, err)
				assert.Nil(t, user)
				assert.Contains(t, err.Error(), "user not found")
			},
		},
		{
			name:   "Database Error",
			msisdn: "+628123456789",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("^SELECT (.+) FROM users WHERE msisdn").
					WithArgs("+628123456789").
					WillReturnError(errors.New("database error"))
			},
			assertFunc: func(t *testing.T, user *models.User, err error) {
				assert.Error(t, err)
				assert.Nil(t, user)
				assert.Contains(t, err.Error(), "failed to get user")
			},
		},
		{
			name:   "Driver Info Query Error",
			msisdn: "+628123456791",
			mockSetup: func(mock sqlmock.Sqlmock) {
				userID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440002")
				rows := sqlmock.NewRows([]string{"id", "msisdn", "fullname", "role", "created_at", "updated_at", "is_active"}).
					AddRow(userID, "+628123456791", "Error Driver", "driver", time.Now(), time.Now(), true)
				mock.ExpectQuery("^SELECT (.+) FROM users WHERE msisdn").
					WithArgs("+628123456791").
					WillReturnRows(rows)

				// Mock driver info query with error
				mock.ExpectQuery("^SELECT \\* FROM drivers WHERE user_id").
					WithArgs(userID).
					WillReturnError(errors.New("driver info error"))
			},
			assertFunc: func(t *testing.T, user *models.User, err error) {
				assert.Error(t, err)
				assert.Nil(t, user)
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
			user, err := repo.GetUserByMSISDN(context.Background(), tc.msisdn)

			// Assert
			tc.assertFunc(t, user, err)

			// Verify all expectations were met
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestCreateUser(t *testing.T) {
	testCases := []struct {
		name       string
		user       models.User
		mockSetup  func(mock sqlmock.Sqlmock)
		assertFunc func(t *testing.T, err error)
	}{
		{
			name: "Success",
			user: models.User{
				MSISDN:   "+628123456789",
				FullName: "John Doe",
				Role:     "user",
				IsActive: true,
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("^INSERT INTO users").
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
				MSISDN:   "+628123456789",
				FullName: "John Doe",
				Role:     "user",
				IsActive: true,
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
			name: "Insert User Error",
			user: models.User{
				MSISDN:   "+628123456789",
				FullName: "John Doe",
				Role:     "user",
				IsActive: true,
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("^INSERT INTO users").
					WillReturnError(errors.New("insert user error"))
				mock.ExpectRollback()
			},
			assertFunc: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to insert user")
			},
		},
		{
			name: "Commit Error",
			user: models.User{
				MSISDN:   "+628123456789",
				FullName: "John Doe",
				Role:     "user",
				IsActive: true,
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("^INSERT INTO users").
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
			err := repo.CreateUser(context.Background(), &tc.user)

			// Assert
			tc.assertFunc(t, err)

			// Check all expectations were met
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestGetUserByField(t *testing.T) {
	testCases := []struct {
		name       string
		field      string
		value      string
		mockSetup  func(mock sqlmock.Sqlmock)
		assertFunc func(t *testing.T, user *models.User, err error)
	}{
		{
			name:  "Success - Regular User",
			field: "id",
			value: "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "msisdn", "fullname", "role", "created_at", "updated_at", "is_active"}).
					AddRow(
						uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
						"+628123456789",
						"John Doe",
						"user",
						time.Now(),
						time.Now(),
						true,
					)
				mock.ExpectQuery("^SELECT \\* FROM users WHERE").
					WithArgs("550e8400-e29b-41d4-a716-446655440000").
					WillReturnRows(rows)
			},
			assertFunc: func(t *testing.T, user *models.User, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, "+628123456789", user.MSISDN)
				assert.Equal(t, "John Doe", user.FullName)
				assert.Equal(t, "user", user.Role)
				assert.Nil(t, user.DriverInfo)
			},
		},
		{
			name:  "Success - Driver User",
			field: "msisdn",
			value: "+628123456790",
			mockSetup: func(mock sqlmock.Sqlmock) {
				userID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")
				rows := sqlmock.NewRows([]string{"id", "msisdn", "fullname", "role", "created_at", "updated_at", "is_active"}).
					AddRow(
						userID,
						"+628123456790",
						"Jane Driver",
						"driver",
						time.Now(),
						time.Now(),
						true,
					)
				mock.ExpectQuery("^SELECT \\* FROM users WHERE").
					WithArgs("+628123456790").
					WillReturnRows(rows)

				// Mock driver info query
				driverRows := sqlmock.NewRows([]string{"user_id", "vehicle_type", "vehicle_plate"}).
					AddRow(userID, "car", "B 1234 ABC")
				mock.ExpectQuery("^SELECT \\* FROM drivers WHERE user_id").
					WithArgs(userID).
					WillReturnRows(driverRows)
			},
			assertFunc: func(t *testing.T, user *models.User, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, "+628123456790", user.MSISDN)
				assert.Equal(t, "Jane Driver", user.FullName)
				assert.Equal(t, "driver", user.Role)
				assert.NotNil(t, user.DriverInfo)
				assert.Equal(t, "car", user.DriverInfo.VehicleType)
			},
		},
		{
			name:  "User Not Found",
			field: "msisdn",
			value: "+628999999999",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("^SELECT \\* FROM users WHERE").
					WithArgs("+628999999999").
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
			user, err := repo.getUserByField(context.Background(), tc.field, tc.value)

			// Assert
			tc.assertFunc(t, user, err)

			// Verify all expectations were met
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
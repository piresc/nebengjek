package repository_test

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/rides/repository"
	"github.com/stretchr/testify/assert"
)

func setupMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New()
	assert.NoError(t, err)
	db := sqlx.NewDb(mockDB, "sqlmock")
	return db, mock
}

func TestCreateRide_Success(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := repository.NewRideRepository(&models.Config{}, db)

	rideID := uuid.New()
	r := &models.Ride{RideID: rideID, DriverID: uuid.New(), CustomerID: uuid.New(), Status: models.RideStatusPending, TotalCost: 0}

	// Expect insert
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO rides")).
		WithArgs(r.RideID, r.DriverID, r.CustomerID, r.Status, r.TotalCost, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	created, err := repo.CreateRide(r)
	assert.NoError(t, err)
	assert.Equal(t, rideID, created.RideID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateTotalCost_NoRows(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := repository.NewRideRepository(&models.Config{}, db)

	rideID := "abc"

	mock.ExpectExec(regexp.QuoteMeta("UPDATE rides")).
		WithArgs(100, rideID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.UpdateTotalCost(context.Background(), rideID, 100)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ride not found")
}

func TestGetRide_Error(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := repository.NewRideRepository(&models.Config{}, db)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT ride_id, driver_id, customer_id")).
		WithArgs("id").
		WillReturnError(assert.AnError)

	_, err := repo.GetRide(context.Background(), "id")
	assert.Error(t, err)
}

func TestCompleteRide_Success(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := repository.NewRideRepository(&models.Config{}, db)

	ride := &models.Ride{RideID: uuid.New()}

	// Expect update marking ride as completed
	mock.ExpectExec(regexp.QuoteMeta("UPDATE rides")).
		WithArgs(ride.RideID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.CompleteRide(context.Background(), ride)
	assert.NoError(t, err)
}

func TestGetBillingLedgerSum_Sum(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := repository.NewRideRepository(&models.Config{}, db)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT SUM(cost)")).
		WithArgs("id").
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(250))

	sum, err := repo.GetBillingLedgerSum(context.Background(), "id")
	assert.NoError(t, err)
	assert.Equal(t, 250, sum)
}

func TestCreatePayment_Success(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := repository.NewRideRepository(&models.Config{}, db)

	pay := &models.Payment{PaymentID: uuid.New(), RideID: uuid.New(), AdjustedCost: 1000, AdminFee: 50, DriverPayout: 950}

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO payments")).
		WithArgs(pay.PaymentID, pay.RideID, pay.AdjustedCost, pay.AdminFee, pay.DriverPayout, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.CreatePayment(context.Background(), pay)
	assert.NoError(t, err)
}

func TestAddBillingEntry_Success(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := repository.NewRideRepository(&models.Config{}, db)

	entry := &models.BillingLedger{EntryID: uuid.New(), RideID: uuid.New(), Distance: 2.5, Cost: 7500}

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO billing_ledger")).
		WithArgs(entry.EntryID, entry.RideID, entry.Distance, entry.Cost, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.AddBillingEntry(context.Background(), entry)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

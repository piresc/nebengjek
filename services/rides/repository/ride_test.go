package repository_test

import (
	"context"
	"regexp"
	"testing"
	"time"

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
	r := &models.Ride{RideID: rideID, DriverID: uuid.New(), PassengerID: uuid.New(), Status: models.RideStatusPending, TotalCost: 0}

	// Expect insert
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO rides")).
		WithArgs(r.RideID, r.DriverID, r.PassengerID, r.Status, r.TotalCost, sqlmock.AnyArg(), sqlmock.AnyArg()).
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

	mock.ExpectQuery(regexp.QuoteMeta("SELECT ride_id, driver_id, passenger_id")).
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
		WithArgs(models.RideStatusCompleted, ride.RideID).
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

	pay := &models.Payment{PaymentID: uuid.New(), RideID: uuid.New(), AdjustedCost: 1000, AdminFee: 50, DriverPayout: 950, Status: models.PaymentStatusPending}

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO payments")).
		WithArgs(pay.PaymentID, pay.RideID, pay.AdjustedCost, pay.AdminFee, pay.DriverPayout, pay.Status, sqlmock.AnyArg()).
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

func TestAddBillingEntry_Error(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := repository.NewRideRepository(&models.Config{}, db)

	entry := &models.BillingLedger{RideID: uuid.New(), Distance: 2.5, Cost: 7500}

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO billing_ledger")).
		WillReturnError(assert.AnError)

	err := repo.AddBillingEntry(context.Background(), entry)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to insert billing entry")
}

func TestUpdateRideStatus_Success(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := repository.NewRideRepository(&models.Config{}, db)

	rideID := uuid.New().String()
	status := models.RideStatusOngoing

	mock.ExpectExec(regexp.QuoteMeta("UPDATE rides")).
		WithArgs(status, rideID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.UpdateRideStatus(context.Background(), rideID, status)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateRideStatus_NotFound(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := repository.NewRideRepository(&models.Config{}, db)

	rideID := uuid.New().String()
	status := models.RideStatusOngoing

	mock.ExpectExec(regexp.QuoteMeta("UPDATE rides")).
		WithArgs(status, rideID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.UpdateRideStatus(context.Background(), rideID, status)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ride not found")
}

func TestGetPaymentByRideID_Success(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := repository.NewRideRepository(&models.Config{}, db)

	rideID := uuid.New().String()
	paymentID := uuid.New()
	rideUUID := uuid.MustParse(rideID)
	createdAt := time.Now()

	rows := sqlmock.NewRows([]string{"payment_id", "ride_id", "adjusted_cost", "admin_fee", "driver_payout", "status", "created_at"}).
		AddRow(paymentID, rideUUID, 8000, 400, 7600, models.PaymentStatusPending, createdAt)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT payment_id, ride_id, adjusted_cost, admin_fee, driver_payout, status, created_at")).
		WithArgs(rideUUID).
		WillReturnRows(rows)

	payment, err := repo.GetPaymentByRideID(context.Background(), rideID)
	assert.NoError(t, err)
	assert.NotNil(t, payment)
	assert.Equal(t, paymentID, payment.PaymentID)
	assert.Equal(t, rideUUID, payment.RideID)
	assert.Equal(t, 8000, payment.AdjustedCost)
	assert.Equal(t, models.PaymentStatusPending, payment.Status)
}

func TestGetPaymentByRideID_NotFound(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := repository.NewRideRepository(&models.Config{}, db)

	rideID := uuid.New().String()
	rideUUID := uuid.MustParse(rideID)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT payment_id, ride_id, adjusted_cost, admin_fee, driver_payout, status, created_at")).
		WithArgs(rideUUID).
		WillReturnError(assert.AnError)

	payment, err := repo.GetPaymentByRideID(context.Background(), rideID)
	assert.Error(t, err)
	assert.Nil(t, payment)
	assert.Contains(t, err.Error(), "failed to get payment for ride")
}

func TestUpdatePaymentStatus_Success(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := repository.NewRideRepository(&models.Config{}, db)

	paymentID := uuid.New().String()
	paymentUUID := uuid.MustParse(paymentID)
	status := models.PaymentStatusAccepted

	mock.ExpectExec(regexp.QuoteMeta("UPDATE payments")).
		WithArgs(status, paymentUUID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.UpdatePaymentStatus(context.Background(), paymentID, status)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdatePaymentStatus_InvalidID(t *testing.T) {
	db, _ := setupMockDB(t)
	repo := repository.NewRideRepository(&models.Config{}, db)

	invalidPaymentID := "invalid-uuid"
	status := models.PaymentStatusAccepted

	err := repo.UpdatePaymentStatus(context.Background(), invalidPaymentID, status)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid payment ID format")
}

func TestGetRide_Success(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := repository.NewRideRepository(&models.Config{}, db)

	rideID := uuid.New().String()
	rideUUID := uuid.MustParse(rideID)
	driverID := uuid.New()
	passengerID := uuid.New()
	createdAt := time.Now()
	updatedAt := time.Now()

	rows := sqlmock.NewRows([]string{"ride_id", "driver_id", "passenger_id", "status", "total_cost", "created_at", "updated_at"}).
		AddRow(rideUUID, driverID, passengerID, models.RideStatusOngoing, 10000, createdAt, updatedAt)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT ride_id, driver_id, passenger_id")).
		WithArgs(rideUUID).
		WillReturnRows(rows)

	ride, err := repo.GetRide(context.Background(), rideID)
	assert.NoError(t, err)
	assert.NotNil(t, ride)
	assert.Equal(t, rideUUID, ride.RideID)
	assert.Equal(t, driverID, ride.DriverID)
	assert.Equal(t, passengerID, ride.PassengerID)
	assert.Equal(t, models.RideStatusOngoing, ride.Status)
	assert.Equal(t, 10000, ride.TotalCost)
}

func TestUpdateTotalCost_Success(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := repository.NewRideRepository(&models.Config{}, db)

	rideID := uuid.New().String()
	additionalCost := 500

	mock.ExpectExec(regexp.QuoteMeta("UPDATE rides")).
		WithArgs(additionalCost, rideID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.UpdateTotalCost(context.Background(), rideID, additionalCost)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCompleteRide_NotFound(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := repository.NewRideRepository(&models.Config{}, db)

	ride := &models.Ride{RideID: uuid.New()}

	mock.ExpectExec(regexp.QuoteMeta("UPDATE rides")).
		WithArgs(models.RideStatusCompleted, ride.RideID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.CompleteRide(context.Background(), ride)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ride not found")
}

func TestGetBillingLedgerSum_NoEntries(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := repository.NewRideRepository(&models.Config{}, db)

	rideID := "test-ride-id"

	mock.ExpectQuery(regexp.QuoteMeta("SELECT SUM(cost)")).
		WithArgs(rideID).
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(0))

	sum, err := repo.GetBillingLedgerSum(context.Background(), rideID)
	assert.NoError(t, err)
	assert.Equal(t, 0, sum)
}

func TestCreatePayment_Error(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := repository.NewRideRepository(&models.Config{}, db)

	payment := &models.Payment{RideID: uuid.New(), AdjustedCost: 1000, AdminFee: 50, DriverPayout: 950}

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO payments")).
		WillReturnError(assert.AnError)

	err := repo.CreatePayment(context.Background(), payment)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create payment")
}

func TestCreateRide_Error(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := repository.NewRideRepository(&models.Config{}, db)

	ride := &models.Ride{DriverID: uuid.New(), PassengerID: uuid.New(), Status: models.RideStatusPending}

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO rides")).
		WillReturnError(assert.AnError)

	created, err := repo.CreateRide(ride)
	assert.Error(t, err)
	assert.Nil(t, created)
}

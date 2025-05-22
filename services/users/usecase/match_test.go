package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/users/mocks" // Corrected mock path
	"github.com/stretchr/testify/assert"
)

// TestHandleCustomerMatchDecision_Confirmed tests the scenario where a customer confirms a match.
func TestHandleCustomerMatchDecision_Confirmed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserGW := mocks.NewMockUserGW(ctrl)
	// Assuming userRepo and cfg are not directly used by HandleCustomerMatchDecision,
	// so they can be nil or also mocked if necessary for NewUserUC.
	// Let's look at NewUserUC signature from services/users/usecase/init.go:
	// func NewUserUC(userRepo users.UserRepo, userGW users.UserGW, cfg *models.Config) *UserUC
	// So, mockUserRepo and a dummy cfg might be needed.
	mockUserRepo := mocks.NewMockUserRepo(ctrl) // Added mockUserRepo
	dummyCfg := &models.Config{}                 // Added dummyCfg

	uc := NewUserUC(mockUserRepo, mockUserGW, dummyCfg)

	ctx := context.Background()
	matchID := "match-confirmed-123"
	driverID := "driver-abc"
	passengerID := "passenger-xyz"

	mp := models.MatchProposal{
		ID:          matchID,
		DriverID:    driverID,
		PassengerID: passengerID,
		MatchStatus: models.MatchStatusAccepted, // Key for this scenario
	}
	natsSubject := constants.SubjectCustomerMatchConfirmed

	// Expected call to the gateway
	mockUserGW.EXPECT().PublishCustomerConfirmedEvent(gomock.Any(), mp).Return(nil).Times(1)

	err := uc.HandleCustomerMatchDecision(ctx, mp, natsSubject)
	assert.NoError(t, err)
}

// TestHandleCustomerMatchDecision_Rejected tests the scenario where a customer rejects a match.
func TestHandleCustomerMatchDecision_Rejected(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserGW := mocks.NewMockUserGW(ctrl)
	mockUserRepo := mocks.NewMockUserRepo(ctrl)
	dummyCfg := &models.Config{}
	uc := NewUserUC(mockUserRepo, mockUserGW, dummyCfg)

	ctx := context.Background()
	matchID := "match-rejected-456"
	driverID := "driver-def"
	passengerID := "passenger-uvw"

	mp := models.MatchProposal{
		ID:          matchID,
		DriverID:    driverID,
		PassengerID: passengerID,
		MatchStatus: models.MatchStatusRejected, // Key for this scenario
	}
	natsSubject := constants.SubjectCustomerMatchRejected

	// Expected call to the gateway
	mockUserGW.EXPECT().PublishCustomerRejectedEvent(gomock.Any(), mp).Return(nil).Times(1)

	err := uc.HandleCustomerMatchDecision(ctx, mp, natsSubject)
	assert.NoError(t, err)
}

// TestHandleCustomerMatchDecision_Mismatch_ConfirmSubject_RejectStatus tests subject/status mismatch.
func TestHandleCustomerMatchDecision_Mismatch_ConfirmSubject_RejectStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserGW := mocks.NewMockUserGW(ctrl)
	mockUserRepo := mocks.NewMockUserRepo(ctrl)
	dummyCfg := &models.Config{}
	uc := NewUserUC(mockUserRepo, mockUserGW, dummyCfg)

	ctx := context.Background()
	mp := models.MatchProposal{
		MatchStatus: models.MatchStatusRejected, // Mismatch
	}
	natsSubject := constants.SubjectCustomerMatchConfirmed

	err := uc.HandleCustomerMatchDecision(ctx, mp, natsSubject)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "natsSubject and MatchStatus mismatch for confirmation")
}

// TestHandleCustomerMatchDecision_Mismatch_RejectSubject_AcceptStatus tests subject/status mismatch.
func TestHandleCustomerMatchDecision_Mismatch_RejectSubject_AcceptStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserGW := mocks.NewMockUserGW(ctrl)
	mockUserRepo := mocks.NewMockUserRepo(ctrl)
	dummyCfg := &models.Config{}
	uc := NewUserUC(mockUserRepo, mockUserGW, dummyCfg)

	ctx := context.Background()
	mp := models.MatchProposal{
		MatchStatus: models.MatchStatusAccepted, // Mismatch
	}
	natsSubject := constants.SubjectCustomerMatchRejected

	err := uc.HandleCustomerMatchDecision(ctx, mp, natsSubject)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "natsSubject and MatchStatus mismatch for rejection")
}

// TestHandleCustomerMatchDecision_UnknownSubject tests handling of an unknown NATS subject.
func TestHandleCustomerMatchDecision_UnknownSubject(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserGW := mocks.NewMockUserGW(ctrl)
	mockUserRepo := mocks.NewMockUserRepo(ctrl)
	dummyCfg := &models.Config{}
	uc := NewUserUC(mockUserRepo, mockUserGW, dummyCfg)

	ctx := context.Background()
	mp := models.MatchProposal{ // Status doesn't matter as much as subject here
		MatchStatus: models.MatchStatusAccepted,
	}
	natsSubject := "some.unknown.subject"

	err := uc.HandleCustomerMatchDecision(ctx, mp, natsSubject)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown nats subject for customer match decision: "+natsSubject)
}

// TestHandleCustomerMatchDecision_GatewayError_Confirmed tests error propagation from gateway on confirmed event.
func TestHandleCustomerMatchDecision_GatewayError_Confirmed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserGW := mocks.NewMockUserGW(ctrl)
	mockUserRepo := mocks.NewMockUserRepo(ctrl)
	dummyCfg := &models.Config{}
	uc := NewUserUC(mockUserRepo, mockUserGW, dummyCfg)

	ctx := context.Background()
	mp := models.MatchProposal{
		MatchStatus: models.MatchStatusAccepted,
	}
	natsSubject := constants.SubjectCustomerMatchConfirmed
	expectedError := errors.New("gateway publish error")

	mockUserGW.EXPECT().PublishCustomerConfirmedEvent(gomock.Any(), mp).Return(expectedError).Times(1)

	err := uc.HandleCustomerMatchDecision(ctx, mp, natsSubject)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, expectedError), "Expected error to wrap gateway error")
	assert.Contains(t, err.Error(), "failed to publish customer match decision")
}

// TestHandleCustomerMatchDecision_GatewayError_Rejected tests error propagation from gateway on rejected event.
func TestHandleCustomerMatchDecision_GatewayError_Rejected(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserGW := mocks.NewMockUserGW(ctrl)
	mockUserRepo := mocks.NewMockUserRepo(ctrl)
	dummyCfg := &models.Config{}
	uc := NewUserUC(mockUserRepo, mockUserGW, dummyCfg)

	ctx := context.Background()
	mp := models.MatchProposal{
		MatchStatus: models.MatchStatusRejected,
	}
	natsSubject := constants.SubjectCustomerMatchRejected
	expectedError := errors.New("gateway publish error")

	mockUserGW.EXPECT().PublishCustomerRejectedEvent(gomock.Any(), mp).Return(expectedError).Times(1)

	err := uc.HandleCustomerMatchDecision(ctx, mp, natsSubject)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, expectedError), "Expected error to wrap gateway error")
	assert.Contains(t, err.Error(), "failed to publish customer match decision")
}

// Note: The HandleCustomerMatchDecision method itself does not perform JSON marshalling;
// it delegates this to the gateway methods (PublishCustomerConfirmedEvent/PublishCustomerRejectedEvent).
// Therefore, a direct test for marshalling failure within HandleCustomerMatchDecision is not applicable.
// Such tests would belong to the gateway method unit tests.

// Also note: The path `github.com/piresc/nebengjek/services/users/mocks` assumes that
// `go generate ./...` or similar has been run for the `users` service to generate mocks
// from `gateways.go` and `repository.go` into that directory.
// If mocks are generated differently (e.g., into a vendor directory or a global mocks directory),
// the import path would need adjustment.
// The current mock path `github.com/piresc/nebengjek/services/users/mocks` is standard for this project.

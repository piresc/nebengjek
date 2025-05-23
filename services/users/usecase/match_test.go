package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/users/mocks" 
	"github.com/stretchr/testify/assert"
)

// TestHandleCustomerMatchDecision_Confirmed tests the scenario where a customer confirms a match.
func TestHandleCustomerMatchDecision_Confirmed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserGW := mocks.NewMockUserGW(ctrl)
	mockUserRepo := mocks.NewMockUserRepo(ctrl) 
	dummyCfg := &models.Config{}                 

	uc := NewUserUC(mockUserRepo, mockUserGW, dummyCfg)

	ctx := context.Background()
	matchID := "match-confirmed-123"
	driverID := "driver-abc"
	passengerID := "passenger-xyz"

	mp := models.MatchProposal{
		ID:          matchID,
		DriverID:    driverID,
		PassengerID: passengerID,
		MatchStatus: models.MatchStatusAccepted, 
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
		MatchStatus: models.MatchStatusRejected, 
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
		ID:          "mismatch-1",
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
		ID:          "mismatch-2",
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
	mp := models.MatchProposal{ 
		ID: "unknown-subject-match",
		MatchStatus: models.MatchStatusAccepted,
	}
	natsSubject := "some.unknown.subject"

	err := uc.HandleCustomerMatchDecision(ctx, mp, natsSubject)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown NATS subject for customer match decision: "+natsSubject)
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
		ID: "gw-err-confirm-1",
		MatchStatus: models.MatchStatusAccepted,
	}
	natsSubject := constants.SubjectCustomerMatchConfirmed
	expectedError := errors.New("gateway publish error")

	mockUserGW.EXPECT().PublishCustomerConfirmedEvent(gomock.Any(), mp).Return(expectedError).Times(1)

	err := uc.HandleCustomerMatchDecision(ctx, mp, natsSubject)
	assert.Error(t, err)
	assert.Equal(t, expectedError, err) 
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
		ID: "gw-err-reject-1",
		MatchStatus: models.MatchStatusRejected,
	}
	natsSubject := constants.SubjectCustomerMatchRejected
	expectedError := errors.New("gateway publish error")

	mockUserGW.EXPECT().PublishCustomerRejectedEvent(gomock.Any(), mp).Return(expectedError).Times(1)

	err := uc.HandleCustomerMatchDecision(ctx, mp, natsSubject)
	assert.Error(t, err)
	assert.Equal(t, expectedError, err) 
}

// TestUpdateBeaconStatus_Success tests successful beacon status update.
func TestUpdateBeaconStatus_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mocks.NewMockUserRepo(ctrl)
	mockUserGW := mocks.NewMockUserGW(ctrl)
	dummyCfg := &models.Config{}
	uc := NewUserUC(mockUserRepo, mockUserGW, dummyCfg)
	
	ctx := context.Background()
	req := &models.BeaconRequest{
		UserID:   "user123",
		IsActive: true,
		Role:     "driver",
		Location: &models.Location{Latitude: 1.0, Longitude: 1.0},
	}

	mockUserGW.EXPECT().PublishBeaconEvent(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	err := uc.UpdateBeaconStatus(ctx, req)
	assert.NoError(t, err)
}

// TestConfirmMatch_Success tests successful match confirmation by a driver.
func TestConfirmMatch_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mocks.NewMockUserRepo(ctrl)
	mockUserGW := mocks.NewMockUserGW(ctrl)
	dummyCfg := &models.Config{}
	uc := NewUserUC(mockUserRepo, mockUserGW, dummyCfg)

	ctx := context.Background()
	driverID := "driver123"
	mp := &models.MatchProposal{
		ID: "matchconfirm-1",
		DriverID: driverID,
		PassengerID: "passenger123",
		MatchStatus: models.MatchStatusPendingCustomerConfirmation,
	}

	mockUserRepo.EXPECT().GetUserByID(ctx, driverID).Return(&models.User{ID: driverID, Role: "driver"}, nil).Times(1)
	mockUserGW.EXPECT().MatchAccept(mp).Return(nil).Times(1)
	
	err := uc.ConfirmMatch(ctx, mp, driverID)
	assert.NoError(t, err)
}

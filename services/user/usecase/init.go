package usecase

import (
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/user"
)

type UserUC struct {
	userRepo user.UserRepo
	UserGW   user.UserGW
	cfg      *models.Config
}

// NewUserUC creates a new user usecase instance
func NewUserUC(
	userRepo user.UserRepo,
	userGW user.UserGW,
	cfg *models.Config,
) *UserUC {
	return &UserUC{
		userRepo: userRepo,
		UserGW:   userGW,
		cfg:      cfg,
	}
}

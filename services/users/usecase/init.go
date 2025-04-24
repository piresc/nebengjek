package usecase

import (
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/users"
)

type UserUC struct {
	userRepo users.UserRepo
	UserGW   users.UserGW
	cfg      *models.Config
}

// NewUserUC creates a new user usecase instance
func NewUserUC(
	userRepo users.UserRepo,
	userGW users.UserGW,
	cfg *models.Config,
) *UserUC {
	return &UserUC{
		userRepo: userRepo,
		UserGW:   userGW,
		cfg:      cfg,
	}
}

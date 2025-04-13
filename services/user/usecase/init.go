package usecase

import (
	"github.com/piresc/nebengjek/services/user/gateways"
	"github.com/piresc/nebengjek/services/user/repository"
)

type UserUC struct {
	userRepo repository.UserRepo
	UserGW   gateways.UserGW
}

// NewUserUC creates a new user usecase instance
func NewUserUC(userRepo repository.UserRepo, natsGW gateways.UserGW) *UserUC {
	return &UserUC{
		userRepo: userRepo,
		UserGW:   natsGW,
	}
}

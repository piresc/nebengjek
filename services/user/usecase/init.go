package usecase

import (
	"github.com/piresc/nebengjek/services/user/gateway"
	"github.com/piresc/nebengjek/services/user/repository"
)

type UserUC struct {
	userRepo repository.UserRepo
	UserGW   gateway.UserGW
}

// NewUserUC creates a new user usecase instance
func NewUserUC(userRepo repository.UserRepo, natsGW gateway.UserGW) *UserUC {
	return &UserUC{
		userRepo: userRepo,
		UserGW:   natsGW,
	}
}

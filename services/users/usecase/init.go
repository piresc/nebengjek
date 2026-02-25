package usecase

import (
	"github.com/piresc/nebengjek/internal/pkg/models/core"
	"github.com/piresc/nebengjek/services/users"
)

type UserUC struct {
	userRepo users.UserRepo
	UserGW   users.UserGW
	cfg      *core.Config
}

// NewUserUC creates a new user usecase instance
func NewUserUC(
	userRepo users.UserRepo,
	userGW users.UserGW,
	cfg *core.Config,
) *UserUC {
	return &UserUC{
		userRepo: userRepo,
		UserGW:   userGW,
		cfg:      cfg,
	}
}

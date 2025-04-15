package repository

import (
	"github.com/jmoiron/sqlx"
	"github.com/piresc/nebengjek/internal/pkg/database"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// UserRepo implements the user repository interface
type UserRepo struct {
	cfg         *models.Config
	db          *sqlx.DB
	redisClient *database.RedisClient
}

// NewUserRepo creates a new user repository instance
func NewUserRepo(cfg *models.Config, db *sqlx.DB, redisClient *database.RedisClient) *UserRepo {
	return &UserRepo{
		cfg:         cfg,
		db:          db,
		redisClient: redisClient,
	}
}

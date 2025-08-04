package service

import (
	"askshop/services/user-service/internal/domain"

	"github.com/gin-gonic/gin"
)

type UserServiceInterface interface {
	GetUserById(ctx *gin.Context, userID string) (*domain.UserModel, error)
	GetUserByEmail(ctx *gin.Context, email string) (*domain.UserModel, error)
	// GetUserMe(ctx *gin.Context, token)
}

type UserService struct {
	userRepo domain.UserRepository
}

func (svc *UserService) GetUserById(ctx *gin.Context, userID string) (*domain.UserModel, error) {
	user, err := svc.userRepo.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return user, nil
}

package service

import (
	"askshop/services/user-service/internal/domain"
	"askshop/services/user-service/pkg/types"
	"askshop/services/user-service/pkg/utils"
	"askshop/shared/auth"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserServiceInterface interface {
	GetUserById(ctx *gin.Context, userID string) (*domain.UserModel, error)
	GetUserByEmail(ctx *gin.Context, email string) (*domain.UserModel, error)
	GetUserByIDOrEmail(ctx *gin.Context, identifier string) (*domain.UserModel, error)
	RegisterUser(ctx *gin.Context, userRequest types.UserRegistrationRequest) (*domain.UserModel, error)
	LoginUser(ctx *gin.Context, loginRequest types.UserLoginRequest) (*domain.UserModel, string, error)
}

type UserService struct {
	userRepo   domain.UserRepository
	JWTservice auth.Manager
}

func NewUserService(userRepo domain.UserRepository) UserServiceInterface {
	return &UserService{
		userRepo: userRepo,
	}
}

func (svc *UserService) GetUserById(ctx *gin.Context, userID string) (*domain.UserModel, error) {
	user, err := svc.userRepo.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (svc *UserService) GetUserByEmail(ctx *gin.Context, email string) (*domain.UserModel, error) {
	user, err := svc.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (svc *UserService) GetUserByIDOrEmail(ctx *gin.Context, identifier string) (*domain.UserModel, error) {
	user, err := svc.userRepo.GetUserByIDOrEmail(ctx, identifier)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (svc *UserService) RegisterUser(ctx *gin.Context, userRequest types.UserRegistrationRequest) (*domain.UserModel, error) {
	//check if user exists already
	_, err := svc.userRepo.GetUserByIDOrEmail(ctx, userRequest.Email)
	if err == nil {
		// User already exists
		return nil, domain.ErrInvalidCredentials
	} else if err != gorm.ErrRecordNotFound {
		// An unexpected error occurred
		return nil, err
	}

	hashedPassword, _ := utils.HashPassword(userRequest.Password)
	// Convert the registration request to a user model
	user := &domain.UserModel{
		FirstName: userRequest.FirstName,
		LastName:  userRequest.LastName,
		Email:     userRequest.Email,
		Password:  hashedPassword,
		Age:       userRequest.Age,
		// Age will be set to 0 as default, can be updated later
	}

	createdUser, err := svc.userRepo.CreateUser(ctx, user)
	if err != nil {
		return nil, err
	}

	return createdUser, nil
}

func (svc *UserService) LoginUser(ctx *gin.Context, loginRequest types.UserLoginRequest) (*domain.UserModel, string, error) {
	// Get user's details
	u, err := svc.userRepo.GetUserByEmail(ctx, loginRequest.Email)
	if err != nil {
		return nil, "", domain.ErrInvalidCredentials
	}
	// check if password is correct
	isPasswordCorrect := utils.CheckPasswordHash(loginRequest.Password, u.Password)
	if !isPasswordCorrect {
		return nil, "", domain.ErrInvalidCredentials
	}
	// generate token
	claims := auth.Claims{
		UserID:    u.ID.String(),
		FirstName: u.FirstName,
		Email:     u.Email,
	}
	at, _, err := svc.JWTservice.GeneratePair(claims)
	if err != nil {
		return nil, "", err
	}
	return u, at, nil
}

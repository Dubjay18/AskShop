package repository

import (
	"askshop/services/user-service/internal/domain"
	"askshop/shared/util"
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// error messages
const (
	ErrUserIDRequired = "UserID is required"
)

type UserRepository struct {
	db *gorm.DB
}

func (ur *UserRepository) CreateUser(ctx *gin.Context, user *domain.UserModel) (*domain.UserModel, error) {
	// Create a new user based on the input or set default values if needed
	newUser := domain.UserModel{
		ID:        uuid.New(),
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Age:       user.Age,
		Email:     user.Email,
		Password:  user.Password,
	}
	// generate random avatar
	newUser.Avatar = util.GetRandomAvatar(newUser.ID)

	result := ur.db.Create(&newUser)
	if result.Error != nil {
		return nil, result.Error
	}

	return &newUser, nil
}

func (ur *UserRepository) GetUser(ctx *gin.Context, userID string) (*domain.UserModel, error) {
	if userID != "" {
		user, err := gorm.G[*domain.UserModel](ur.db).Where("id = ?", userID).First(ctx)
		if err != nil {
			return nil, err
		}
		return user, nil
	}
	return nil, errors.New(ErrUserIDRequired)
}

func (ur *UserRepository) GetUserByEmail(ctx *gin.Context, email string) (*domain.UserModel, error) {
	if email == "" {
		return nil, errors.New("email is required")
	}

	user, err := gorm.G[*domain.UserModel](ur.db).Where("email = ?", email).First(ctx)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (ur *UserRepository) GetUserByIDOrEmail(ctx *gin.Context, identifier string) (*domain.UserModel, error) {
	if identifier == "" {
		return nil, errors.New("identifier (ID or email) is required")
	}

	// Check if identifier looks like an email (contains @ symbol)
	if strings.Contains(identifier, "@") {
		return ur.GetUserByEmail(ctx, identifier)
	}

	// Otherwise, treat it as an ID
	return ur.GetUser(ctx, identifier)
}
func NewUserRepository(db *gorm.DB) domain.UserRepository {
	return &UserRepository{
		db: db,
	}
}

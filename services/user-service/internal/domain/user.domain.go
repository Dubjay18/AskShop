package domain

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")
)

type UserModel struct {
	gorm.Model
	ID        uuid.UUID `json:"id" gorm:"primaryKey"`
	FirstName string    `json:"firstName" gorm:"first_name"`
	LastName  string    `json:"lastName" gorm:"last_name"`
	Age       int64     `json:"age" gorm:"age"`
	Email     string    `json:"email" gorm:"email"`
	Password  string    `json:"-" gorm:"password"`
	Avatar    string    `json:"avatar" gorm:"avatar"`
}
type UserRepository interface {
	CreateUser(ctx *gin.Context, user *UserModel) (*UserModel, error)
	GetUser(ctx *gin.Context, userID string) (*UserModel, error)
	GetUserByEmail(ctx *gin.Context, email string) (*UserModel, error)
	GetUserByIDOrEmail(ctx *gin.Context, identifier string) (*UserModel, error)
}

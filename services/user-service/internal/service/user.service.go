package service

import (
	"askshop/services/user-service/internal/domain"
	"askshop/services/user-service/pkg/types"
	"askshop/services/user-service/pkg/utils"
	"askshop/shared/auth"
	"askshop/shared/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserServiceInterface interface {
	GetUserById(ctx *gin.Context, userID string) (*domain.UserModel, error)
	GetUserByEmail(ctx *gin.Context, email string) (*domain.UserModel, error)
	GetUserByIDOrEmail(ctx *gin.Context, identifier string) (*domain.UserModel, error)
	RegisterUser(ctx *gin.Context, userRequest types.UserRegistrationRequest) (*domain.UserModel, error)
	RegisterUserWithExternalID(ctx *gin.Context, userRequest types.UserRegistrationRequest, externalID string) (*domain.UserModel, error)
	LoginUser(ctx *gin.Context, loginRequest types.UserLoginRequest) (*domain.UserModel, string, error)
}

type UserService struct {
	userRepo   domain.UserRepository
	JWTservice *auth.Manager
	logger     *logger.Logger
}

func NewUserService(userRepo domain.UserRepository) UserServiceInterface {
	return &UserService{
		userRepo:   userRepo,
		JWTservice: auth.NewManager(auth.LoadConfig()),
		logger:     logger.Default("user-service"),
	}
}

func (svc *UserService) GetUserById(ctx *gin.Context, userID string) (*domain.UserModel, error) {
	ctxLogger := logger.FromGin(ctx)
	ctxLogger.Debugf("Getting user by ID: %s", userID)

	user, err := svc.userRepo.GetUser(ctx, userID)
	if err != nil {
		ctxLogger.Errorf("Failed to get user by ID %s: %v", userID, err)
		return nil, err
	}

	ctxLogger.Infof("Successfully retrieved user by ID: %s", userID)
	return user, nil
}

func (svc *UserService) GetUserByEmail(ctx *gin.Context, email string) (*domain.UserModel, error) {
	ctxLogger := logger.FromGin(ctx)
	ctxLogger.Debugf("Getting user by email: %s", email)

	user, err := svc.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		ctxLogger.Errorf("Failed to get user by email %s: %v", email, err)
		return nil, err
	}

	ctxLogger.Infof("Successfully retrieved user by email: %s", email)
	return user, nil
}

func (svc *UserService) GetUserByIDOrEmail(ctx *gin.Context, identifier string) (*domain.UserModel, error) {
	ctxLogger := logger.FromGin(ctx)
	ctxLogger.Debugf("Getting user by ID or email: %s", identifier)

	user, err := svc.userRepo.GetUserByIDOrEmail(ctx, identifier)
	if err != nil {
		ctxLogger.Errorf("Failed to get user by ID or email %s: %v", identifier, err)
		return nil, err
	}

	ctxLogger.Infof("Successfully retrieved user by ID or email: %s", identifier)
	return user, nil
}

func (svc *UserService) RegisterUser(ctx *gin.Context, userRequest types.UserRegistrationRequest) (*domain.UserModel, error) {
	ctxLogger := logger.FromGin(ctx)
	ctxLogger.Debugf("Registering new user with email: %s", userRequest.Email)

	// Check if user already exists
	_, err := svc.userRepo.GetUserByEmail(ctx, userRequest.Email)
	if err == nil {
		ctxLogger.Warnf("User with email %s already exists", userRequest.Email)
		return nil, domain.ErrUserAlreadyExists
	}
	// Check if error is not a not found error
	if err != gorm.ErrRecordNotFound {
		ctxLogger.Errorf("Error checking existing user by email %s: %v", userRequest.Email, err)
		return nil, err
	}

	// Create user model from request
	user := &domain.UserModel{
		FirstName: userRequest.FirstName,
		LastName:  userRequest.LastName,
		Email:     userRequest.Email,
		Age:       userRequest.Age,
	}

	// Hash the password
	ctxLogger.Debug("Hashing user password")
	hashedPassword, err := utils.HashPassword(userRequest.Password)
	if err != nil {
		ctxLogger.Errorf("Failed to hash password for user %s: %v", userRequest.Email, err)
		return nil, err
	}
	user.Password = hashedPassword

	// Save user
	ctxLogger.Debug("Saving user to database")
	createdUser, err := svc.userRepo.CreateUser(ctx, user)
	if err != nil {
		ctxLogger.Errorf("Failed to create user %s: %v", userRequest.Email, err)
		return nil, err
	}

	ctxLogger.Infof("Successfully registered user with email: %s", userRequest.Email)
	return createdUser, nil
}

// RegisterUserWithExternalID registers a user using an external auth provider's id
// The password is not stored locally when using an external provider.
func (svc *UserService) RegisterUserWithExternalID(ctx *gin.Context, userRequest types.UserRegistrationRequest, externalID string) (*domain.UserModel, error) {
	ctxLogger := logger.FromGin(ctx)
	ctxLogger.Debugf("Registering user with external ID. Email: %s, ExternalID: %s", userRequest.Email, externalID)

	// check if user exists already by email
	existingUser, err := svc.userRepo.GetUserByIDOrEmail(ctx, userRequest.Email)
	if err == nil {
		ctxLogger.Warnf("User with email %s already exists", userRequest.Email)
		// Return the existing user instead of an error when registering with external ID
		ctxLogger.Info("Returning existing user for external auth registration")
		return existingUser, nil
	} else if err != gorm.ErrRecordNotFound {
		ctxLogger.Errorf("Error checking existing user by email %s: %v", userRequest.Email, err)
		return nil, err
	}

	// Build user model. Use externalID as the ID if provided.
	user := &domain.UserModel{
		FirstName: userRequest.FirstName,
		LastName:  userRequest.LastName,
		Email:     userRequest.Email,
		Age:       userRequest.Age,
		Password:  "", // password managed by external provider
	}

	if externalID != "" {
		ctxLogger.Debug("Attempting to use external ID as user UUID")
		// Attempt to set UUID from externalID
		if uid, err := uuid.Parse(externalID); err == nil {
			user.ID = uid
		} else {
			ctxLogger.Warnf("Could not parse external ID as UUID: %v", err)
		}
	}

	ctxLogger.Debug("Creating user with external ID")
	createdUser, err := svc.userRepo.CreateUser(ctx, user)
	if err != nil {
		ctxLogger.Errorf("Failed to create user with external ID: %v", err)
		return nil, err
	}

	ctxLogger.Infof("Successfully registered user with external ID. Email: %s", userRequest.Email)
	return createdUser, nil
}

func (svc *UserService) LoginUser(ctx *gin.Context, loginRequest types.UserLoginRequest) (*domain.UserModel, string, error) {
	ctxLogger := logger.FromGin(ctx)
	ctxLogger.Debugf("Attempting login for user with email: %s", loginRequest.Email)

	// Get user's details
	u, err := svc.userRepo.GetUserByEmail(ctx, loginRequest.Email)
	if err != nil {
		ctxLogger.Warnf("Login failed - user not found with email: %s", loginRequest.Email)
		return nil, "", domain.ErrInvalidCredentials
	}

	// check if password is correct
	ctxLogger.Debug("Verifying password")
	isPasswordCorrect := utils.CheckPasswordHash(loginRequest.Password, u.Password)
	if !isPasswordCorrect {
		ctxLogger.Warnf("Login failed - incorrect password for user: %s", loginRequest.Email)
		return nil, "", domain.ErrInvalidCredentials
	}

	// generate token
	ctxLogger.Debug("Generating JWT token")
	claims := auth.Claims{
		UserID:    u.ID.String(),
		FirstName: u.FirstName,
		Email:     u.Email,
	}
	at, _, err := svc.JWTservice.GeneratePair(claims)
	if err != nil {
		ctxLogger.Errorf("Failed to generate token for user %s: %v", loginRequest.Email, err)
		return nil, "", err
	}

	ctxLogger.Infof("User logged in successfully: %s", loginRequest.Email)
	return u, at, nil
}

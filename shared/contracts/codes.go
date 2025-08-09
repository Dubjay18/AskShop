package contracts

// Common error codes used across services
const (
	CodeInvalidCredentials  = "INVALID_CREDENTIALS"
	CodeInternalServerError = "INTERNAL_SERVER_ERROR"
	CodeInvalidRequestBody  = "INVALID_REQUEST_BODY"
)

// User-related error codes
const (
	CodeUserAlreadyExists      = "USER_ALREADY_EXISTS"
	CodeUserNotFound           = "USER_NOT_FOUND"
	CodeUserIDRequired         = "USER_ID_REQUIRED"
	CodeEmailRequired          = "EMAIL_REQUIRED"
	CodeUserIdentifierRequired = "USER_IDENTIFIER_REQUIRED"
)

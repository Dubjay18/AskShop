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

// Auth-related error codes
const (
	CodeAuthInvalidBody          = "AUTH_INVALID_BODY"
	CodeAuthRegisterFailed       = "AUTH_REGISTER_FAILED"
	CodeAuthUserExists           = "AUTH_USER_EXISTS"
	CodeAuthInvalidCredentials   = "AUTH_INVALID_CREDENTIALS"
	CodeAuthRefreshFailed        = "AUTH_REFRESH_FAILED"
	CodeAuthLogoutFailed         = "AUTH_LOGOUT_FAILED"
	CodeAuthProfileNotFound      = "AUTH_PROFILE_NOT_FOUND"
	CodeAuthProfileUpdateFailed  = "AUTH_PROFILE_UPDATE_FAILED"
	CodeAuthProfileConflict      = "AUTH_PROFILE_CONFLICT"
	CodeAuthPasswordChangeFailed = "AUTH_PASSWORD_CHANGE_FAILED"
	CodeAuthInvalidOldPassword   = "AUTH_INVALID_OLD_PASSWORD"
	CodeAuthHeaderRequired       = "AUTH_HEADER_REQUIRED"
	CodeAuthInvalidToken         = "AUTH_INVALID_TOKEN"
)

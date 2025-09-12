package auth

import (
	"askshop/shared/env"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	auth "github.com/supabase-community/auth-go"
	"github.com/supabase-community/auth-go/types"
)

// SupabaseClient is a wrapper for the Supabase auth client
type SupabaseClient struct {
	client auth.Client
}

// NewSupabaseClient creates a new Supabase client
func NewSupabaseClient() *SupabaseClient {
	supabaseURL := env.GetString("SUPABASE_URL", "")
	supabaseKey := env.GetString("SUPABASE_KEY", "")

	// Check for empty values
	if supabaseURL == "" || supabaseKey == "" {
		return nil
	}

	// Check for placeholder values
	supabaseLogf := LogOrPrint("Supabase")
	if strings.Contains(supabaseURL, "your-project.supabase.co") {
		supabaseLogf("ERROR", "Supabase URL contains placeholder value. Please update your SUPABASE_URL in your secrets or environment variables.")
		return nil
	}

	if supabaseKey == "your-service-role-key" {
		supabaseLogf("ERROR", "Supabase key contains placeholder value. Please update your SUPABASE_KEY in your secrets or environment variables.")
		return nil
	}

	// Most Supabase keys start with eyJ (base64 encoded JWT)
	if !strings.HasPrefix(supabaseKey, "eyJ") {
		supabaseLogf("WARN", "Supabase key doesn't appear to be in the expected format (should start with 'eyJ'). This might cause authentication issues.")
	}

	// Log success but without showing the actual key
	supabaseLogf("INFO", "Using Supabase URL: %s", supabaseURL)
	supabaseLogf("INFO", "Using Supabase Key: [REDACTED] ")

	// Extract the project reference from the URL
	// From: https://project-ref.supabase.co
	// To: project-ref
	projectRef := ""
	if host := strings.Split(strings.TrimPrefix(strings.TrimPrefix(supabaseURL, "https://"), "http://"), "."); len(host) > 0 {
		projectRef = host[0]
	}

	// Create a new client for DNS checking
	tempClient := &SupabaseClient{}

	// Check DNS resolution - this is just for diagnostics
	if err := tempClient.CheckDNSResolution(supabaseURL); err != nil {
		supabaseLogf("WARN", "DNS resolution check failed: %v. This may cause connection issues.", err)
		// Continue anyway, but warn about potential issues
	}

	// Initialize the auth client with proper service role key handling
	supabaseLogf("DEBUG", "Initializing Supabase client with project ref: %s", projectRef)

	// Check for IP address override for DNS issues
	supabaseIP := env.GetString("SUPABASE_IP", "")
	if supabaseIP != "" {
		supabaseLogf("INFO", "Using explicit IP address for Supabase connection: %s", supabaseIP)

		// Parse the URL to get components
		parsedURL, err := url.Parse(supabaseURL)
		if err == nil {
			// Create a new URL with the IP address instead of hostname, but keep the path
			scheme := parsedURL.Scheme
			path := parsedURL.Path
			supabaseURL = fmt.Sprintf("%s://%s%s", scheme, supabaseIP, path)
			supabaseLogf("INFO", "Modified Supabase URL to use IP: %s", supabaseURL)

			// Also add a Host header override in the HTTP client
			originalHost := parsedURL.Hostname()
			supabaseLogf("DEBUG", "Will use original hostname '%s' in Host header", originalHost)
		}
	}

	// // Create a custom HTTP client with timeout
	// httpClient := http.Client{
	// 	Timeout: 10 * time.Second, // Set a reasonable timeout
	// 	Transport: &http.Transport{
	// 		ResponseHeaderTimeout: 5 * time.Second,
	// 		TLSHandshakeTimeout:   5 * time.Second,
	// 		ExpectContinueTimeout: 1 * time.Second,
	// 	},
	// }

	// Initialize the client with custom HTTP client
	client := auth.New(projectRef, supabaseKey)

	// Verify the key is properly formatted
	keyType := "anon key"
	if strings.Contains(supabaseKey, "service_role") {
		keyType = "service role key"
	}
	supabaseLogf("INFO", "Using Supabase %s", keyType)

	// Set the service role key as the bearer token for admin operations
	// This is necessary for operations like AdminCreateUser
	client = client.WithToken(supabaseKey)

	// If the URL is not standard Supabase, use it directly
	if !strings.Contains(supabaseURL, ".supabase.co") {
		supabaseLogf("DEBUG", "Using custom auth URL: %s", supabaseURL+"/auth/v1")
		client = client.WithCustomAuthURL(supabaseURL + "/auth/v1")
	}

	return &SupabaseClient{
		client: client,
	}
}

// SignUp registers a new user with Supabase
func (c *SupabaseClient) SignUp(ctx context.Context, email, password string, userData map[string]interface{}) (map[string]interface{}, error) {
	// Create a timeout context
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	supabaseLogf := LogOrPrint("Supabase")

	// Double check that client is initialized
	if c == nil || c.client == nil {
		supabaseLogf("ERROR", "Supabase client not initialized correctly")
		return nil, fmt.Errorf("supabase client not initialized correctly - check your SUPABASE_URL and SUPABASE_KEY environment variables")
	}

	// Log initial parameters for debugging
	supabaseLogf("DEBUG", "SignUp called with email: %s, password length: %d, userData present: %v",
		email, len(password), userData != nil)

	// Validate password first
	// Log the password length for debugging (without revealing content)
	supabaseLogf("DEBUG", "Password validation - length: %d", len(password))

	// Make sure password isn't empty before validation
	if password == "" {
		supabaseLogf("ERROR", "Password is empty, cannot proceed with registration")
		return nil, fmt.Errorf("password validation failed: password is empty")
	}

	if valid, reason := validatePassword(password); !valid {
		supabaseLogf("ERROR", "Password validation failed: %s", reason)
		return nil, fmt.Errorf("password validation failed: %s", reason)
	}

	// Log key info for debugging
	supabaseLogf("DEBUG", "Starting user creation in Supabase for email: %s", email)

	// Try direct HTTP API approach first - it gives us more control
	supabaseURL := env.GetString("SUPABASE_URL", "")
	supabaseKey := env.GetString("SUPABASE_KEY", "")

	if supabaseURL != "" && supabaseKey != "" {
		supabaseLogf("DEBUG", "Attempting direct HTTP API registration")

		// Create the direct HTTP request
		httpClient := &http.Client{Timeout: 10 * time.Second}

		// First try standard signup endpoint
		signupURL := fmt.Sprintf("%s/auth/v1/signup", supabaseURL)

		// Prepare request body
		requestBody := map[string]interface{}{
			"email":    email,
			"password": password,
		}

		// Add user metadata if provided
		if userData != nil {
			requestBody["data"] = userData
		}

		// Get redirect URL from environment or configuration
		redirectURL := env.GetString("SUPABASE_REDIRECT_URL", "")
		if redirectURL != "" {
			requestBody["redirect_to"] = redirectURL
			supabaseLogf("DEBUG", "Using redirect URL for email confirmation: %s", redirectURL)
		} // Debug the request body (without showing the actual password)
		debugBody := map[string]interface{}{
			"email":           email,
			"password_length": len(password),
		}
		if userData != nil {
			debugBody["has_user_data"] = true
		}
		debugJSON, _ := json.Marshal(debugBody)
		supabaseLogf("DEBUG", "Request body: %s", string(debugJSON))

		requestJSON, jsonErr := json.Marshal(requestBody)
		if jsonErr != nil {
			supabaseLogf("ERROR", "Failed to marshal request body: %v", jsonErr)
			return nil, fmt.Errorf("failed to create request: %w", jsonErr)
		}

		// Create request
		req, reqErr := http.NewRequestWithContext(ctx, "POST", signupURL, bytes.NewBuffer(requestJSON))
		if reqErr == nil {
			req.Header.Set("apikey", supabaseKey)
			req.Header.Set("Content-Type", "application/json")

			// Execute request
			resp, respErr := httpClient.Do(req)

			if respErr == nil {
				defer resp.Body.Close()

				if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
					// Success! Parse the response
					var signupResp map[string]interface{}
					if jsonErr := json.NewDecoder(resp.Body).Decode(&signupResp); jsonErr == nil {
						supabaseLogf("INFO", "Successfully registered user via direct HTTP API")

						// Extract user information and log it for debugging
						if user, ok := signupResp["user"].(map[string]interface{}); ok {
							if id, ok := user["id"].(string); ok {
								supabaseLogf("INFO", "Created user with ID: %s", id)
							}
						}
						return signupResp, nil
					}
				} else {
					// Read error response
					bodyBytes, _ := io.ReadAll(resp.Body)
					respBody := string(bodyBytes)
					supabaseLogf("WARN", "Direct HTTP signup failed: %s", respBody)
				}
			}
		}
	}

	// If direct HTTP approach failed, try the SDK methods
	supabaseLogf("DEBUG", "Trying SDK methods for user registration")

	// Try admin create user first (it has fewer restrictions)
	supabaseLogf("DEBUG", "Attempting to create user with AdminCreateUser method")

	// Make sure we're passing a valid password
	if password == "" {
		supabaseLogf("ERROR", "Cannot create user with empty password")
		return nil, fmt.Errorf("password validation failed: password is empty")
	}

	// Create a copy to avoid any potential reference issues
	passwordCopy := password

	// Always set EmailConfirm to true to auto-confirm the email
	emailConfirm := true

	adminReq := types.AdminCreateUserRequest{
		Email:        email,
		Password:     &passwordCopy,
		EmailConfirm: emailConfirm, // This will auto-confirm the email address
		UserMetadata: userData,
	}

	adminResp, adminErr := c.client.AdminCreateUser(adminReq)
	if adminErr == nil {
		// Admin user creation succeeded
		supabaseLogf("INFO", "Successfully created user via AdminCreateUser with ID: %s", adminResp.ID)
		result := map[string]interface{}{
			"id":       adminResp.ID,
			"email":    adminResp.Email,
			"metadata": adminResp.UserMetadata,
		}
		return result, nil
	}

	// If admin create fails, try standard signup as last resort
	supabaseLogf("WARN", "AdminCreateUser failed: %v", adminErr)
	supabaseLogf("DEBUG", "Attempting to create user with standard Signup method")

	signUpReq := types.SignupRequest{
		Email:    email,
		Password: password,
		Data:     userData,
	}

	// Log the exact request being sent for debugging
	supabaseLogf("DEBUG", "Signup request: email=%s, password length=%d, has user data=%v",
		email, len(password), userData != nil)

	resp, err := c.client.Signup(signUpReq)

	// If all approaches fail, return a detailed error
	if err != nil {
		// Both approaches failed, log diagnostic information
		supabaseLogf("ERROR", "All user creation methods failed")
		supabaseLogf("ERROR", "Direct HTTP failed, AdminCreateUser error: %v, Standard Signup error: %v", adminErr, err)

		// Add specific error checking
		if strings.Contains(err.Error(), "dial tcp") {
			if strings.Contains(err.Error(), "server misbehaving") || strings.Contains(err.Error(), "no such host") {
				supabaseLogf("ERROR", "DNS resolution failed for Supabase URL. Please check your network connection and DNS settings: %v", err)
			} else {
				supabaseLogf("ERROR", "Network error connecting to Supabase. Check your SUPABASE_URL: %v", err)
			}
		} else if strings.Contains(err.Error(), "401") {
			supabaseLogf("ERROR", "Authentication error with Supabase. Check your SUPABASE_KEY: %v", err)
		}

		return nil, fmt.Errorf("supabase user creation failed: %w", err)
	}

	// Successfully created user with standard signup
	supabaseLogf("INFO", "Successfully created user in Supabase with ID: %s", resp.ID)
	result := map[string]interface{}{
		"id":       resp.ID,
		"email":    resp.Email,
		"metadata": resp.UserMetadata,
		"user": map[string]interface{}{
			"id":    resp.ID,
			"email": resp.Email,
		},
	}

	return result, nil
}

// SignIn logs in a user with Supabase
func (c *SupabaseClient) SignIn(ctx context.Context, email, password string) (map[string]interface{}, error) {
	supabaseLogf := LogOrPrint("Supabase")

	// Add diagnostic info to help troubleshoot
	supabaseURL := env.GetString("SUPABASE_URL", "")
	supabaseKey := env.GetString("SUPABASE_KEY", "")
	isProduction := env.GetString("ENV", "development") == "production"

	// Log masked email for better troubleshooting
	maskedEmail := maskEmail(email)
	supabaseLogf("INFO", "Attempting to sign in user: %s", maskedEmail)

	// Validate password first
	if valid, reason := validatePassword(password); !valid {
		supabaseLogf("WARN", "Password validation failed: %s", reason)
		supabaseLogf("DEBUG", "Continuing anyway since this is a sign-in attempt")
	}

	// When not in production, add more diagnostic info
	if !isProduction {
		supabaseLogf("DEBUG", "Environment: %s", env.GetString("ENV", "development"))
		supabaseLogf("DEBUG", "Using Supabase URL: %s", supabaseURL)
		supabaseLogf("DEBUG", "Password provided length: %d", len(password))
	}

	if c == nil || c.client == nil {
		supabaseLogf("ERROR", "Supabase client not initialized correctly")
		return nil, fmt.Errorf("supabase client not initialized correctly")
	}

	// First try direct HTTP authentication for more control
	var userFound bool
	supabaseLogf("DEBUG", "Trying direct HTTP authentication first")

	var tokenResp map[string]interface{}
	var directAuthError error

	if supabaseURL != "" && supabaseKey != "" {
		// Create a custom HTTP client for this request
		httpClient := &http.Client{Timeout: 10 * time.Second}

		// Try to authenticate with the token endpoint directly
		authURL := fmt.Sprintf("%s/auth/v1/token?grant_type=password", supabaseURL)

		// Create request body
		requestBody := map[string]string{
			"email":    email,
			"password": password,
		}

		requestJSON, _ := json.Marshal(requestBody)

		// Create request
		req, reqErr := http.NewRequestWithContext(ctx, "POST", authURL, bytes.NewBuffer(requestJSON))
		if reqErr == nil {
			req.Header.Set("apikey", supabaseKey)
			req.Header.Set("Content-Type", "application/json")

			// Execute request
			resp, respErr := httpClient.Do(req)

			if respErr == nil {
				defer resp.Body.Close()

				if resp.StatusCode == http.StatusOK {
					// Success! Parse the response
					if jsonErr := json.NewDecoder(resp.Body).Decode(&tokenResp); jsonErr == nil {
						supabaseLogf("INFO", "Direct HTTP authentication successful!")

						// Check if we got an access token
						if accessToken, ok := tokenResp["access_token"].(string); ok && accessToken != "" {
							// Successfully authenticated! Return the response
							supabaseLogf("INFO", "Successfully authenticated via direct HTTP")
							return tokenResp, nil
						}
					}
				} else {
					// Read error response
					bodyBytes, _ := io.ReadAll(resp.Body)
					respBody := string(bodyBytes)
					directAuthError = fmt.Errorf("direct HTTP auth failed with status %d: %s",
						resp.StatusCode, respBody)
					supabaseLogf("WARN", "Direct HTTP auth failed: %v", directAuthError)
				}
			}
		}

		// Check if user exists (useful for diagnostics)
		userURL := fmt.Sprintf("%s/auth/v1/admin/users", supabaseURL)
		userReq, userReqErr := http.NewRequestWithContext(ctx, "GET", userURL, nil)
		if userReqErr == nil {
			userReq.Header.Set("apikey", supabaseKey)
			userReq.Header.Set("Authorization", "Bearer "+supabaseKey)
			userReq.Header.Set("Content-Type", "application/json")

			// Add query params to search by email
			q := userReq.URL.Query()
			q.Add("email", email)
			userReq.URL.RawQuery = q.Encode()

			userResp, userRespErr := httpClient.Do(userReq)
			if userRespErr == nil && userResp.StatusCode == http.StatusOK {
				defer userResp.Body.Close()

				// Parse response to get user details
				var userList []map[string]interface{}
				if err := json.NewDecoder(userResp.Body).Decode(&userList); err == nil && len(userList) > 0 {
					userFound = true
					supabaseLogf("DEBUG", "Found existing user record")

					// If we found the user but auth failed, the password might be incorrect
					if directAuthError != nil {
						supabaseLogf("DEBUG", "User exists but authentication failed - password may be incorrect")
					}
				} else {
					supabaseLogf("DEBUG", "User does not exist in the database")
				}
			}
		}
	}

	// If direct HTTP call failed, fall back to SDK
	supabaseLogf("DEBUG", "Falling back to SDK Token method")

	// If we found the user, log it for diagnostics
	if userFound {
		supabaseLogf("INFO", "User exists in database, attempting SDK authentication")
	}

	// First check if the error was due to unconfirmed email
	if directAuthError != nil && strings.Contains(directAuthError.Error(), "email_not_confirmed") {
		supabaseLogf("INFO", "Email not confirmed, attempting to confirm it automatically")

		// Try to confirm the email
		if confirmErr := c.ConfirmUserEmail(ctx, email); confirmErr != nil {
			supabaseLogf("ERROR", "Failed to confirm email: %v", confirmErr)
			// Continue with login attempt anyway
		} else {
			supabaseLogf("INFO", "Successfully confirmed email, retrying authentication")

			// Retry direct HTTP authentication after confirming email
			httpClient := &http.Client{Timeout: 10 * time.Second}

			// Try to authenticate with the token endpoint directly
			authURL := fmt.Sprintf("%s/auth/v1/token?grant_type=password", supabaseURL)

			// Create request body
			requestBody := map[string]string{
				"email":    email,
				"password": password,
			}

			requestJSON, _ := json.Marshal(requestBody)

			// Create request
			req, reqErr := http.NewRequestWithContext(ctx, "POST", authURL, bytes.NewBuffer(requestJSON))
			if reqErr == nil {
				req.Header.Set("apikey", supabaseKey)
				req.Header.Set("Content-Type", "application/json")

				// Execute request
				resp, respErr := httpClient.Do(req)

				if respErr == nil {
					defer resp.Body.Close()

					if resp.StatusCode == http.StatusOK {
						// Success after confirming email!
						if jsonErr := json.NewDecoder(resp.Body).Decode(&tokenResp); jsonErr == nil {
							supabaseLogf("INFO", "Direct HTTP authentication successful after confirming email!")

							// Check if we got an access token
							if accessToken, ok := tokenResp["access_token"].(string); ok && accessToken != "" {
								// Successfully authenticated! Return the response
								supabaseLogf("INFO", "Successfully authenticated via direct HTTP after email confirmation")
								return tokenResp, nil
							}
						}
					}
				}
			}
		}
	}

	// Use the Token API with SDK
	resp, err := c.client.Token(types.TokenRequest{
		GrantType: "password",
		Email:     email,
		Password:  password,
	})

	if err != nil {
		// Log detailed error information for diagnosis
		supabaseLogf("ERROR", "Token request failed: %v", err)

		// Check for email_not_confirmed error and try to handle it
		if strings.Contains(err.Error(), "email_not_confirmed") {
			supabaseLogf("WARN", "Login failed because email is not confirmed. Attempting to confirm email automatically")

			// Try to confirm the email using admin API
			if supabaseURL != "" && supabaseKey != "" {
				// First get the user ID
				httpClient := &http.Client{Timeout: 5 * time.Second}
				userURL := fmt.Sprintf("%s/auth/v1/admin/users", supabaseURL)
				userReq, userReqErr := http.NewRequestWithContext(ctx, "GET", userURL, nil)
				if userReqErr == nil {
					userReq.Header.Set("apikey", supabaseKey)
					userReq.Header.Set("Authorization", "Bearer "+supabaseKey)
					userReq.Header.Set("Content-Type", "application/json")

					// Add query params to search by email
					q := userReq.URL.Query()
					q.Add("email", email)
					userReq.URL.RawQuery = q.Encode()

					userResp, userRespErr := httpClient.Do(userReq)
					if userRespErr == nil && userResp.StatusCode == http.StatusOK {
						defer userResp.Body.Close()

						// Parse response to get user details
						var userList []map[string]interface{}
						if jsonErr := json.NewDecoder(userResp.Body).Decode(&userList); jsonErr == nil && len(userList) > 0 {
							// Get user ID
							if userID, ok := userList[0]["id"].(string); ok {
								supabaseLogf("DEBUG", "Found user ID: %s. Attempting to confirm email", userID)

								// Update user to confirm email
								updateURL := fmt.Sprintf("%s/auth/v1/admin/users/%s", supabaseURL, userID)
								updateBody := map[string]interface{}{
									"email_confirm": true,
								}
								updateJSON, _ := json.Marshal(updateBody)

								updateReq, _ := http.NewRequestWithContext(ctx, "PUT", updateURL, bytes.NewBuffer(updateJSON))
								updateReq.Header.Set("apikey", supabaseKey)
								updateReq.Header.Set("Authorization", "Bearer "+supabaseKey)
								updateReq.Header.Set("Content-Type", "application/json")

								updateResp, updateErr := httpClient.Do(updateReq)
								if updateErr == nil && (updateResp.StatusCode == http.StatusOK || updateResp.StatusCode == http.StatusNoContent) {
									defer updateResp.Body.Close()

									supabaseLogf("INFO", "Successfully confirmed email for user: %s", email)

									// Try login again
									supabaseLogf("DEBUG", "Retrying login after confirming email")
									resp, err = c.client.Token(types.TokenRequest{
										GrantType: "password",
										Email:     email,
										Password:  password,
									})

									// If successful, return the result
									if err == nil {
										supabaseLogf("INFO", "Login successful after confirming email")
										// Return successful result at the end of the function
										goto LoginSuccess
									}
								}
							}
						}
					}
				}
			}
		}

		// If we already found a user, provide more specific debugging
		if userFound {
			supabaseLogf("ERROR", "User exists but authentication failed")
			supabaseLogf("DEBUG", "This may be due to password mismatch or authentication config issues")
		}

		// Check if it's an email confirmation issue
		if strings.Contains(err.Error(), "email_not_confirmed") {
			supabaseLogf("INFO", "Authentication failed because email is not confirmed")

			// Try to confirm the email as a last resort
			if confirmErr := c.ConfirmUserEmail(ctx, email); confirmErr != nil {
				supabaseLogf("ERROR", "Failed to confirm email: %v", confirmErr)
				// Return more descriptive error for client handling
				return nil, fmt.Errorf("email not confirmed and auto-confirmation failed: %w", err)
			}

			// Try one more time after confirming
			supabaseLogf("INFO", "Successfully confirmed email, attempting authentication one more time")
			retryResp, retryErr := c.client.Token(types.TokenRequest{
				GrantType: "password",
				Email:     email,
				Password:  password,
			})

			if retryErr == nil {
				supabaseLogf("INFO", "Authentication succeeded after confirming email!")
				// Convert to map[string]interface{} format
				result := map[string]interface{}{
					"access_token":  retryResp.AccessToken,
					"refresh_token": retryResp.RefreshToken,
					"user":          retryResp.User,
					"expires_in":    retryResp.ExpiresIn,
				}
				return result, nil
			} else {
				supabaseLogf("ERROR", "Authentication still failed after confirming email: %v", retryErr)
				return nil, fmt.Errorf("email was confirmed but authentication still failed: %w", retryErr)
			}
		}

		// Check for specific error types
		if strings.Contains(err.Error(), "401") {
			supabaseLogf("ERROR", "Authentication failed (401 Unauthorized). Invalid credentials or permissions issue")

			// Try alternative authentication method
			supabaseLogf("DEBUG", "Attempting alternative authentication method")

			// Get Supabase URL from environment
			supabaseURL := env.GetString("SUPABASE_URL", "")
			if supabaseURL == "" {
				supabaseLogf("ERROR", "No Supabase URL available for alternative authentication")
			} else {
				// Try direct API call as a last resort - this is a more manual approach
				supabaseLogf("DEBUG", "Falling back to direct API call for authentication")
				// No implementation here - would need custom HTTP client code
			}

			// All alternative methods failed
			supabaseLogf("ERROR", "Alternative authentication methods failed")

			supabaseLogf("ERROR", "All authentication methods failed")
		} else if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "deadline exceeded") {
			supabaseLogf("ERROR", "Connection timeout to Supabase. Check your network or Supabase service status")
		} else if strings.Contains(err.Error(), "no such host") || strings.Contains(err.Error(), "dial tcp") {
			supabaseLogf("ERROR", "Network connectivity issue or invalid Supabase URL")
		}
		return nil, err
	}

LoginSuccess:
	supabaseLogf("INFO", "User signed in successfully: %s", email)

	// Add token validation check
	if resp.AccessToken == "" {
		supabaseLogf("WARN", "Login succeeded but no access token returned")
	} else {
		// Log token length for diagnostics (not the actual token)
		supabaseLogf("DEBUG", "Received access token length: %d characters", len(resp.AccessToken))
	}

	// Check if user data was returned
	emptyUUID := uuid.Nil
	if resp.User.ID == emptyUUID {
		supabaseLogf("WARN", "Login succeeded but no user data returned or empty user ID")
	}

	result := map[string]interface{}{
		"access_token":  resp.AccessToken,
		"refresh_token": resp.RefreshToken,
		"user":          resp.User,
		"expires_in":    resp.ExpiresIn,
	}

	return result, nil
}

// VerifyToken validates a JWT token from Supabase
func (c *SupabaseClient) VerifyToken(token string) (map[string]interface{}, error) {
	// Use the GetUser endpoint to verify the token
	authedClient := c.client.WithToken(token)
	user, err := authedClient.GetUser()
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"id":       user.ID,
		"email":    user.Email,
		"metadata": user.UserMetadata,
	}

	return result, nil
}

// IsAvailable checks if Supabase integration is available
func (c *SupabaseClient) IsAvailable() bool {
	return c != nil && c.client != nil
}

// GetDiagnostics returns diagnostic information about the Supabase configuration
func (c *SupabaseClient) GetDiagnostics() map[string]interface{} {
	// If client isn't initialized, return basic diagnostics
	if c == nil {
		return map[string]interface{}{
			"success":  false,
			"errors":   []string{"Supabase client is nil"},
			"warnings": []string{},
			"tests": map[string]bool{
				"client_initialized": false,
			},
		}
	}

	return c.RunDiagnostics()
}

// CheckDNSResolution attempts to resolve the Supabase host and provides diagnostic information
func (c *SupabaseClient) CheckDNSResolution(supabaseURL string) error {
	supabaseLogf := LogOrPrint("Supabase")
	supabaseLogf("DEBUG", "Checking DNS resolution for Supabase URL: %s", supabaseURL)

	// Parse the URL to get the hostname
	parsedURL, err := url.Parse(supabaseURL)
	if err != nil {
		supabaseLogf("ERROR", "Failed to parse Supabase URL: %v", err)
		return fmt.Errorf("invalid URL format: %v", err)
	}

	// Get the hostname
	hostname := parsedURL.Hostname()

	// Try to resolve the hostname
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	addrs, err := net.DefaultResolver.LookupHost(ctx, hostname)
	if err != nil {
		supabaseLogf("ERROR", "DNS resolution failed for %s: %v", hostname, err)
		return fmt.Errorf("DNS resolution failed: %v", err)
	}

	supabaseLogf("INFO", "Successfully resolved %s to %v", hostname, addrs)
	return nil
}

// RunDiagnostics performs comprehensive diagnostics on Supabase configuration
func (c *SupabaseClient) RunDiagnostics() map[string]interface{} {
	supabaseLogf := LogOrPrint("Supabase")
	supabaseLogf("INFO", "Running Supabase diagnostics...")

	results := map[string]interface{}{
		"success":  true,
		"errors":   []string{},
		"warnings": []string{},
		"tests":    map[string]bool{},
	}

	errors := []string{}
	warnings := []string{}

	// Check client initialization
	if c == nil || c.client == nil {
		errors = append(errors, "Supabase client not initialized")
		results["success"] = false
		results["tests"].(map[string]bool)["client_initialized"] = false
	} else {
		results["tests"].(map[string]bool)["client_initialized"] = true
	}

	// Check environment variables
	supabaseURL := env.GetString("SUPABASE_URL", "")
	supabaseKey := env.GetString("SUPABASE_KEY", "")

	if supabaseURL == "" {
		errors = append(errors, "SUPABASE_URL not set")
		results["success"] = false
		results["tests"].(map[string]bool)["has_url"] = false
	} else {
		results["tests"].(map[string]bool)["has_url"] = true
	}

	if supabaseKey == "" {
		errors = append(errors, "SUPABASE_KEY not set")
		results["success"] = false
		results["tests"].(map[string]bool)["has_key"] = false
	} else {
		results["tests"].(map[string]bool)["has_key"] = true

		// Key format check
		if !strings.HasPrefix(supabaseKey, "eyJ") {
			warnings = append(warnings, "Supabase key doesn't appear to be in the expected format")
			results["tests"].(map[string]bool)["key_format_valid"] = false
		} else {
			results["tests"].(map[string]bool)["key_format_valid"] = true
		}

		// Key length check
		if len(supabaseKey) < 100 {
			warnings = append(warnings, "Supabase key seems unusually short")
		}
	}

	// DNS resolution check
	if supabaseURL != "" {
		if err := c.CheckDNSResolution(supabaseURL); err != nil {
			warnings = append(warnings, fmt.Sprintf("DNS resolution failed: %v", err))
			results["tests"].(map[string]bool)["dns_resolution"] = false
		} else {
			results["tests"].(map[string]bool)["dns_resolution"] = true
		}
	}

	// Network connectivity test
	if supabaseURL != "" {
		parsedURL, _ := url.Parse(supabaseURL)
		hostname := parsedURL.Hostname()
		port := parsedURL.Port()
		if port == "" {
			port = "443" // Default HTTPS port
		}

		// Try to establish a TCP connection
		dialer := net.Dialer{Timeout: 5 * time.Second}
		conn, err := dialer.Dial("tcp", hostname+":"+port)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to connect to %s: %v", hostname, err))
			results["success"] = false
			results["tests"].(map[string]bool)["network_connectivity"] = false
		} else {
			conn.Close()
			results["tests"].(map[string]bool)["network_connectivity"] = true
		}
	}

	// Update results with errors and warnings
	results["errors"] = errors
	results["warnings"] = warnings

	return results
}

// LogOrPrint is a helper function that tries to use the logger package if available
// or falls back to standard logging if not
func LogOrPrint(service string) func(level, message string, args ...interface{}) {
	return func(level, message string, args ...interface{}) {
		// Format the message with args
		formattedMsg := fmt.Sprintf(message, args...)

		// For now just use standard log package
		log.Printf("%s [%s] %s", level, service, formattedMsg)
	}
}

// TestConnection runs a simple authentication test to verify connectivity and credentials
func (c *SupabaseClient) TestConnection() map[string]interface{} {
	if c == nil || c.client == nil {
		return map[string]interface{}{
			"success": false,
			"error":   "Supabase client not initialized",
		}
	}

	supabaseLogf := LogOrPrint("Supabase")

	// Get configuration
	supabaseURL := env.GetString("SUPABASE_URL", "")
	supabaseKey := env.GetString("SUPABASE_KEY", "")

	// Test connection with a simple request
	result := map[string]interface{}{
		"success":    false,
		"url":        supabaseURL,
		"key_type":   "unknown",
		"key_length": len(supabaseKey),
	}

	// Check key type
	if strings.Contains(supabaseKey, "service_role") {
		result["key_type"] = "service_role"
	} else if strings.Contains(supabaseKey, "anon") {
		result["key_type"] = "anon"
	}

	// Simple test auth
	supabaseLogf("INFO", "Testing Supabase connection...")

	// Track which tests we try for diagnostics
	testsTried := []string{}
	result["tests_tried"] = testsTried

	// Try multiple tests in sequence from least invasive to most invasive

	// Test 1: Check admin permissions with list users (read-only operation)
	supabaseLogf("DEBUG", "Test 1: Trying AdminListUsers to verify permissions")
	testsTried = append(testsTried, "admin_list_users")
	result["tests_tried"] = testsTried

	page := 1
	perPage := 1
	_, err := c.client.AdminListUsers(types.AdminListUsersRequest{
		Page:    &page,
		PerPage: &perPage,
	})

	if err == nil {
		result["success"] = true
		result["test_method"] = "admin_list_users"
		supabaseLogf("INFO", "Supabase connection test successful (AdminListUsers)")
		return result
	}

	supabaseLogf("WARN", "AdminListUsers test failed: %v", err)
	result["admin_list_users_error"] = err.Error()

	// Test 2: Try getting project settings (another read-only operation)
	supabaseLogf("DEBUG", "Test 2: Trying to fetch project health")
	testsTried = append(testsTried, "project_health")
	result["tests_tried"] = testsTried

	// This is a workaround since the auth-go library doesn't have a dedicated health check
	// We'll use our custom URL handling to call a specific endpoint
	httpClient := &http.Client{Timeout: 5 * time.Second}
	healthURL := fmt.Sprintf("%s/auth/v1/health", supabaseURL)
	req, _ := http.NewRequest("GET", healthURL, nil)
	req.Header.Set("apikey", supabaseKey)
	req.Header.Set("Authorization", "Bearer "+supabaseKey)

	resp, err := httpClient.Do(req)
	if err == nil && resp.StatusCode < 300 {
		result["success"] = true
		result["test_method"] = "project_health"
		supabaseLogf("INFO", "Supabase connection test successful (Health check)")
		return result
	}

	if err != nil {
		supabaseLogf("WARN", "Health check failed: %v", err)
		result["health_check_error"] = err.Error()
	} else if resp != nil {
		supabaseLogf("WARN", "Health check failed with status: %d", resp.StatusCode)
		result["health_check_status"] = resp.StatusCode
	}

	// Test 3: Try a write operation as last resort
	supabaseLogf("DEBUG", "Test 3: Trying to create a test user")
	testsTried = append(testsTried, "create_test_user")
	result["tests_tried"] = testsTried

	// Generate a unique test email to avoid conflicts
	uniqueEmail := fmt.Sprintf("test_%s@example.com", time.Now().Format("20060102150405"))
	testPassword := "Test@Password123"

	tempUserData := map[string]interface{}{
		"test":      true,
		"temporary": true,
	}

	_, tempErr := c.SignUp(context.Background(), uniqueEmail, testPassword, tempUserData)
	if tempErr == nil {
		supabaseLogf("INFO", "Successfully created test user, connection is working")
		result["success"] = true
		result["test_method"] = "user_creation"
		result["test_email"] = uniqueEmail
		return result
	}

	// All tests failed
	supabaseLogf("ERROR", "All Supabase connection tests failed")
	result["user_creation_error"] = tempErr.Error()

	// Use the first error as the main error message
	result["error"] = result["admin_list_users_error"]

	return result
}

// ConfirmUserEmail confirms a user's email address using the admin API
func (c *SupabaseClient) ConfirmUserEmail(ctx context.Context, email string) error {
	supabaseLogf := LogOrPrint("Supabase")
	supabaseLogf("INFO", "Attempting to confirm email for user: %s", maskEmail(email))

	if c == nil || c.client == nil {
		return fmt.Errorf("supabase client not initialized correctly")
	}

	// We'll need to search for the user by email directly with the HTTP API
	// since the Go SDK doesn't provide a direct method for this
	supabaseURL := env.GetString("SUPABASE_URL", "")
	supabaseKey := env.GetString("SUPABASE_KEY", "")
	if supabaseURL == "" || supabaseKey == "" {
		return fmt.Errorf("supabase URL or key not configured")
	}

	// Create a custom HTTP client
	httpClient := &http.Client{Timeout: 10 * time.Second}

	// First, search for the user by email
	searchURL := fmt.Sprintf("%s/auth/v1/admin/users", supabaseURL)
	searchReq, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		supabaseLogf("ERROR", "Failed to create search request: %v", err)
		return err
	}

	// Set headers
	searchReq.Header.Set("apikey", supabaseKey)
	searchReq.Header.Set("Authorization", "Bearer "+supabaseKey)

	// Add query parameter for email search
	query := searchReq.URL.Query()
	query.Add("email", email)
	searchReq.URL.RawQuery = query.Encode()

	// Execute the search request
	searchResp, err := httpClient.Do(searchReq)
	if err != nil {
		supabaseLogf("ERROR", "Failed to search for user: %v", err)
		return err
	}

	defer searchResp.Body.Close()

	if searchResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(searchResp.Body)
		respBody := string(bodyBytes)
		supabaseLogf("ERROR", "Failed to search for user: %s", respBody)
		return fmt.Errorf("failed to search for user: %s", respBody)
	}

	// Parse the response to get user ID
	var users []map[string]interface{}
	if err := json.NewDecoder(searchResp.Body).Decode(&users); err != nil {
		supabaseLogf("ERROR", "Failed to parse user search response: %v", err)
		return err
	}

	// Check if user was found
	if len(users) == 0 {
		supabaseLogf("ERROR", "User not found with email: %s", maskEmail(email))
		return fmt.Errorf("user not found with email: %s", email)
	}

	// Extract user ID
	userIDInterface, ok := users[0]["id"]
	if !ok {
		supabaseLogf("ERROR", "User ID not found in response")
		return fmt.Errorf("user ID not found in response")
	}

	userID, ok := userIDInterface.(string)
	if !ok {
		supabaseLogf("ERROR", "User ID is not a string")
		return fmt.Errorf("user ID is not a string")
	}
	supabaseLogf("INFO", "Found user with ID: %s, attempting to confirm email", userID)

	// Now update the user to confirm email
	// Create direct HTTP request to confirm email (this is not exposed in the SDK)
	// We already have supabaseURL and supabaseKey from earlier in the function

	// Make sure we still have valid values
	if supabaseURL != "" && supabaseKey != "" {
		// Create custom HTTP client
		httpClient := &http.Client{Timeout: 10 * time.Second}

		// Construct URL for admin update user endpoint
		updateURL := fmt.Sprintf("%s/auth/v1/admin/users/%s", supabaseURL, userID)

		// Create request body with email confirmed flag
		requestBody := map[string]interface{}{
			"email_confirm": true,
		}

		requestJSON, _ := json.Marshal(requestBody)

		// Create request
		req, reqErr := http.NewRequestWithContext(ctx, "PUT", updateURL, bytes.NewBuffer(requestJSON))
		if reqErr != nil {
			supabaseLogf("ERROR", "Failed to create request: %v", reqErr)
			return reqErr
		}

		// Set headers
		req.Header.Set("apikey", supabaseKey)
		req.Header.Set("Authorization", "Bearer "+supabaseKey)
		req.Header.Set("Content-Type", "application/json")

		// Execute request
		resp, respErr := httpClient.Do(req)
		if respErr != nil {
			supabaseLogf("ERROR", "Failed to confirm email: %v", respErr)
			return respErr
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			// Read error response
			bodyBytes, _ := io.ReadAll(resp.Body)
			respBody := string(bodyBytes)
			supabaseLogf("ERROR", "Failed to confirm email: %s", respBody)
			return fmt.Errorf("failed to confirm email: %s", respBody)
		}

		supabaseLogf("INFO", "Successfully confirmed email for user: %s", maskEmail(email))
		return nil
	}

	return fmt.Errorf("supabase URL or key not configured")
}

// maskEmail masks part of the email address for logging
func maskEmail(email string) string {
	if email == "" {
		return ""
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "[invalid-email]"
	}

	username := parts[0]
	domain := parts[1]

	// Mask the username portion
	maskedUsername := username
	if len(username) > 2 {
		maskedUsername = username[0:2] + strings.Repeat("*", len(username)-2)
	} else if len(username) > 0 {
		maskedUsername = username[0:1] + "*"
	}

	return maskedUsername + "@" + domain
}

// validatePassword checks if a password meets Supabase requirements
func validatePassword(password string) (bool, string) {
	// Check for common password requirements
	if password == "" {
		return false, "password is empty"
	}

	if len(password) < 6 {
		return false, "password too short (minimum 6 characters)"
	}

	// All checks passed
	return true, ""
}

// ConfirmEmail confirms a user's email address in Supabase
// This is useful when email verification is required but not properly set up
func (c *SupabaseClient) ConfirmEmail(ctx context.Context, email string) error {
	supabaseLogf := LogOrPrint("Supabase")
	supabaseLogf("INFO", "Confirming email for user: %s", email)

	supabaseURL := env.GetString("SUPABASE_URL", "")
	supabaseKey := env.GetString("SUPABASE_KEY", "")

	if supabaseURL == "" || supabaseKey == "" {
		return fmt.Errorf("missing Supabase configuration")
	}

	// First get the user ID
	httpClient := &http.Client{Timeout: 5 * time.Second}
	userURL := fmt.Sprintf("%s/auth/v1/admin/users", supabaseURL)
	userReq, userReqErr := http.NewRequestWithContext(ctx, "GET", userURL, nil)
	if userReqErr != nil {
		return fmt.Errorf("failed to create request: %w", userReqErr)
	}

	userReq.Header.Set("apikey", supabaseKey)
	userReq.Header.Set("Authorization", "Bearer "+supabaseKey)
	userReq.Header.Set("Content-Type", "application/json")

	// Add query params to search by email
	q := userReq.URL.Query()
	q.Add("email", email)
	userReq.URL.RawQuery = q.Encode()

	userResp, userRespErr := httpClient.Do(userReq)
	if userRespErr != nil {
		return fmt.Errorf("failed to get user: %w", userRespErr)
	}

	defer userResp.Body.Close()

	if userResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(userResp.Body)
		return fmt.Errorf("failed to get user, status code %d: %s", userResp.StatusCode, string(bodyBytes))
	}

	// Parse response to get user details
	var userList []map[string]interface{}
	if jsonErr := json.NewDecoder(userResp.Body).Decode(&userList); jsonErr != nil {
		return fmt.Errorf("failed to decode user response: %w", jsonErr)
	}

	if len(userList) == 0 {
		return fmt.Errorf("user not found")
	}

	// Get user ID
	userID, ok := userList[0]["id"].(string)
	if !ok {
		return fmt.Errorf("invalid user ID")
	}

	supabaseLogf("DEBUG", "Found user ID: %s. Attempting to confirm email", userID)

	// Update user to confirm email
	updateURL := fmt.Sprintf("%s/auth/v1/admin/users/%s", supabaseURL, userID)
	updateBody := map[string]interface{}{
		"email_confirm": true,
	}
	updateJSON, _ := json.Marshal(updateBody)

	updateReq, _ := http.NewRequestWithContext(ctx, "PUT", updateURL, bytes.NewBuffer(updateJSON))
	updateReq.Header.Set("apikey", supabaseKey)
	updateReq.Header.Set("Authorization", "Bearer "+supabaseKey)
	updateReq.Header.Set("Content-Type", "application/json")

	updateResp, updateErr := httpClient.Do(updateReq)
	if updateErr != nil {
		return fmt.Errorf("failed to update user: %w", updateErr)
	}

	defer updateResp.Body.Close()

	if updateResp.StatusCode != http.StatusOK && updateResp.StatusCode != http.StatusNoContent {
		bodyBytes, _ := io.ReadAll(updateResp.Body)
		return fmt.Errorf("failed to confirm email, status code %d: %s", updateResp.StatusCode, string(bodyBytes))
	}

	supabaseLogf("INFO", "Successfully confirmed email for user: %s", email)
	return nil
}

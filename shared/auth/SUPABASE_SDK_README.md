# Supabase SDK Integration for AskShop

This integration adds the official Supabase Go SDK for authentication to the AskShop platform.

## What's Added

1. **Supabase Auth SDK**: Implemented the official `github.com/supabase-community/auth-go` SDK for more reliable and maintainable Supabase authentication.

2. **Custom Client Wrapper**: Created a simplified wrapper client in `shared/auth/supabase_client.go` that provides high-level functions:
   - `SignUp` - Register new users
   - `SignIn` - Authenticate users and return tokens
   - `VerifyToken` - Validate tokens and return user information

3. **Middleware Integration**: Updated the authentication middleware to use the SDK for token verification.

## Setup Instructions

1. **Install the Supabase SDK**:
   ```bash
   go get github.com/supabase-community/auth-go
   ```

2. **Environment Variables**:
   Make sure your `.env` file contains:
   ```
   SUPABASE_URL=your_supabase_url
   SUPABASE_KEY=your_supabase_service_key
   ```

3. **Usage in Code**:
   ```go
   supaClient := supabaseAuth.NewSupabaseClient()
   if !supaClient.IsAvailable() {
       // Handle error - Supabase not configured
   }
   
   // Sign up a user
   userMetadata := map[string]interface{}{
       "first_name": "John",
       "last_name": "Doe",
   }
   user, err := supaClient.SignUp(ctx, "user@example.com", "password123", userMetadata)
   
   // Sign in a user
   loginResult, err := supaClient.SignIn(ctx, "user@example.com", "password123")
   token := loginResult["access_token"].(string)
   
   // Verify a token
   userInfo, err := supaClient.VerifyToken(token)
   ```

## Benefits Over Previous Implementation

1. **Maintained SDK**: Using the official Supabase SDK ensures compatibility with API changes and security updates.

2. **Better Error Handling**: The SDK provides more consistent error handling and typesafe responses.

3. **Simplified Authentication Flow**: Reduces boilerplate code and improves readability.

4. **Future-Proof**: Easier to update as Supabase releases new features.

## Next Steps

1. Run `go mod tidy` to ensure all dependencies are properly updated.

2. Update your Postman collection auth scripts to work with the Supabase token structure.

3. Test login and registration flows to verify the SDK integration works as expected.
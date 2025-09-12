package test

import (
	"askshop/shared/auth"
	"context"
	"fmt"
	"os"
	"time"
)

func main() {
	// Create unique email for testing
	uniqueEmail := fmt.Sprintf("test_%d@example.com", time.Now().Unix())
	password := "Test123456" // Strong password with uppercase, lowercase and numbers

	// Create Supabase client
	client := auth.NewSupabaseClient()
	if client == nil {
		fmt.Println("Failed to initialize Supabase client. Check your environment variables.")
		os.Exit(1)
	}

	// First test - check client availability
	if !client.IsAvailable() {
		fmt.Println("Supabase client reports as not available")
		os.Exit(1)
	}

	fmt.Println("Supabase client initialized successfully")

	// Run diagnostics
	fmt.Println("\n=== Running Supabase Diagnostics ===")
	diagnostics := client.GetDiagnostics()
	fmt.Printf("Diagnostics success: %v\n", diagnostics["success"])
	if errors, ok := diagnostics["errors"].([]string); ok && len(errors) > 0 {
		fmt.Println("Errors:")
		for _, err := range errors {
			fmt.Printf("- %s\n", err)
		}
	}
	if warnings, ok := diagnostics["warnings"].([]string); ok && len(warnings) > 0 {
		fmt.Println("Warnings:")
		for _, warn := range warnings {
			fmt.Printf("- %s\n", warn)
		}
	}

	// Test registration
	fmt.Println("\n=== Testing User Registration ===")
	fmt.Printf("Registering test user with email: %s\n", uniqueEmail)
	userData := map[string]interface{}{
		"first_name": "Test",
		"last_name":  "User",
		"test":       true,
	}

	ctx := context.Background()
	registerResp, registerErr := client.SignUp(ctx, uniqueEmail, password, userData)

	if registerErr != nil {
		fmt.Printf("Registration FAILED: %v\n", registerErr)
		os.Exit(1)
	}

	fmt.Println("Registration SUCCESS!")
	fmt.Printf("Response: %+v\n", registerResp)

	// Test login
	fmt.Println("\n=== Testing User Login ===")
	fmt.Printf("Logging in with email: %s\n", uniqueEmail)

	loginResp, loginErr := client.SignIn(ctx, uniqueEmail, password)
	if loginErr != nil {
		fmt.Printf("Login FAILED: %v\n", loginErr)
		os.Exit(1)
	}

	fmt.Println("Login SUCCESS!")
	fmt.Printf("Access Token Length: %d\n", len(loginResp["access_token"].(string)))
	fmt.Printf("Response: %+v\n", loginResp)

	fmt.Println("\n=== All Tests Passed Successfully ===")
}

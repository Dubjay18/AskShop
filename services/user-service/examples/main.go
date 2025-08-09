package main

import (
	"fmt"
	"log"
)

// This is a simple example showing how to use the flexible user lookup
func main() {
	fmt.Println("User Service - Flexible User Lookup Examples")
	fmt.Println("===========================================")

	fmt.Println("\nAvailable endpoints:")
	fmt.Println("1. GET /api/users/123e4567-e89b-12d3-a456-426614174000")
	fmt.Println("   - Lookup user by UUID")

	fmt.Println("2. GET /api/users/user@example.com")
	fmt.Println("   - Lookup user by email (automatically detected)")

	fmt.Println("3. GET /api/users/by-id/123e4567-e89b-12d3-a456-426614174000")
	fmt.Println("   - Lookup user by ID specifically")

	fmt.Println("4. GET /api/users/by-email?email=user@example.com")
	fmt.Println("   - Lookup user by email specifically")

	fmt.Println("\nThe main benefit of /api/users/:identifier is that it can handle both:")
	fmt.Println("- UUIDs (detected as ID)")
	fmt.Println("- Email addresses (detected by presence of '@' symbol)")

	fmt.Println("\nExample usage in frontend:")
	fmt.Println("fetch('/api/users/' + userIdentifier) // works for both ID and email")

	log.Println("User service supports flexible user lookup by ID or email")
}

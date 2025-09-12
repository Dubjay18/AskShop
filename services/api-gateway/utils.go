// This file contains helper functions to convert between different data types in the API gateway

package main

import (
	"encoding/json"
	"fmt"
)

// ConvertUserResponse converts an untyped response map to a User struct
// This helps handle variations in ID types (string vs numeric) coming from services
func ConvertUserResponse(data interface{}) (User, error) {
	var user User

	// First try to convert to map
	userMap := make(map[string]interface{})

	switch v := data.(type) {
	case map[string]interface{}:
		userMap = v
	default:
		// Try to marshal and unmarshal to convert between types
		jsonData, err := json.Marshal(data)
		if err != nil {
			return user, fmt.Errorf("failed to marshal user data: %w", err)
		}
		if err := json.Unmarshal(jsonData, &userMap); err != nil {
			return user, fmt.Errorf("failed to unmarshal user data to map: %w", err)
		}
	}

	// Handle ID field specially since it might be a number or string
	if id, exists := userMap["id"]; exists {
		switch v := id.(type) {
		case string:
			user.ID = v
		case float64:
			user.ID = fmt.Sprintf("%.0f", v)
		case int:
			user.ID = fmt.Sprintf("%d", v)
		case int64:
			user.ID = fmt.Sprintf("%d", v)
		default:
			// Try to convert to string
			jsonID, err := json.Marshal(id)
			if err != nil {
				return user, fmt.Errorf("failed to convert ID to string: %w", err)
			}
			// Remove quotes if it was already a string
			idStr := string(jsonID)
			if len(idStr) > 2 && idStr[0] == '"' && idStr[len(idStr)-1] == '"' {
				user.ID = idStr[1 : len(idStr)-1]
			} else {
				user.ID = idStr
			}
		}
	}

	// Handle other fields
	if firstName, exists := userMap["firstName"]; exists {
		if str, ok := firstName.(string); ok {
			user.FirstName = str
		}
	}

	if lastName, exists := userMap["lastName"]; exists {
		if str, ok := lastName.(string); ok {
			user.LastName = str
		}
	}

	if email, exists := userMap["email"]; exists {
		if str, ok := email.(string); ok {
			user.Email = str
		}
	}

	if age, exists := userMap["age"]; exists {
		switch v := age.(type) {
		case float64:
			user.Age = int(v)
		case int:
			user.Age = v
		case int64:
			user.Age = int(v)
		}
	}

	// If name doesn't exist but first and last name do, combine them
	if user.Name == "" && user.FirstName != "" && user.LastName != "" {
		user.Name = user.FirstName + " " + user.LastName
	}

	return user, nil
}

// ConvertAuthResponse converts an untyped response map to an AuthResponse struct
func ConvertAuthResponse(data interface{}) (AuthResponse, error) {
	var authResp AuthResponse

	// First try to convert to map
	respMap := make(map[string]interface{})

	switch v := data.(type) {
	case map[string]interface{}:
		respMap = v
	default:
		// Try to marshal and unmarshal to convert between types
		jsonData, err := json.Marshal(data)
		if err != nil {
			return authResp, fmt.Errorf("failed to marshal auth response: %w", err)
		}
		if err := json.Unmarshal(jsonData, &respMap); err != nil {
			return authResp, fmt.Errorf("failed to unmarshal auth response to map: %w", err)
		}
	}

	// Handle user field
	if userData, exists := respMap["user"]; exists {
		user, err := ConvertUserResponse(userData)
		if err != nil {
			return authResp, fmt.Errorf("failed to convert user data: %w", err)
		}
		authResp.User = user
	}

	// Handle token fields
	if accessToken, exists := respMap["accessToken"]; exists {
		if str, ok := accessToken.(string); ok {
			authResp.AccessToken = str
		}
	}

	if refreshToken, exists := respMap["refreshToken"]; exists {
		if str, ok := refreshToken.(string); ok {
			authResp.RefreshToken = str
		}
	}

	return authResp, nil
}

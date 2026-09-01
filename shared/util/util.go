package util

import (
	"fmt"

	"github.com/google/uuid"
)

// GetRandomAvatar returns a random avatar URL from the randomuser.me API
func GetRandomAvatar(index uuid.UUID) string {
	return fmt.Sprintf("https://randomuser.me/api/portraits/lego/%s.jpg", index.String())
}

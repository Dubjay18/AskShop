package util

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestGetRandomAvatar(t *testing.T) {
	id := uuid.New()
	got := GetRandomAvatar(id)

	want := "https://randomuser.me/api/portraits/lego/" + id.String() + ".jpg"
	if got != want {
		t.Errorf("GetRandomAvatar(%s) = %q, want %q", id, got, want)
	}
	if !strings.HasPrefix(got, "https://randomuser.me/api/portraits/lego/") {
		t.Errorf("GetRandomAvatar(%s) = %q, missing expected prefix", id, got)
	}
	if !strings.HasSuffix(got, ".jpg") {
		t.Errorf("GetRandomAvatar(%s) = %q, missing .jpg suffix", id, got)
	}
}

func TestGetRandomAvatarNilUUID(t *testing.T) {
	got := GetRandomAvatar(uuid.Nil)
	want := "https://randomuser.me/api/portraits/lego/00000000-0000-0000-0000-000000000000.jpg"
	if got != want {
		t.Errorf("GetRandomAvatar(uuid.Nil) = %q, want %q", got, want)
	}
}

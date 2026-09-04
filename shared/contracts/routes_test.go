package contracts

import "testing"

func TestJoinPaths(t *testing.T) {
	tests := []struct {
		name  string
		parts []string
		want  string
	}{
		{"no parts", nil, ""},
		{"single part", []string{"/api/v1"}, "/api/v1"},
		{"joins without duplicate slash", []string{"/api/v1", "/products"}, "/api/v1/products"},
		{"handles missing leading slash", []string{"/api/v1", "products"}, "/api/v1/products"},
		{"handles trailing slash on left", []string{"/api/v1/", "/products"}, "/api/v1/products"},
		{"handles both slashes present", []string{"/api/v1/", "/products/"}, "/api/v1/products/"},
		{"three parts", []string{"/api/v1", "/products", "/:id"}, "/api/v1/products/:id"},
		{"all empty parts resolve to root", []string{"", ""}, "/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JoinPaths(tt.parts...)
			if got != tt.want {
				t.Errorf("JoinPaths(%v) = %q, want %q", tt.parts, got, tt.want)
			}
		})
	}
}

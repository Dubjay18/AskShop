package service

import "testing"

func TestClampPagination(t *testing.T) {
	tests := []struct {
		name         string
		page         int
		pageSize     int
		wantPage     int
		wantPageSize int
	}{
		{"defaults are valid", 1, 20, 1, 20},
		{"zero page clamps to 1", 0, 20, 1, 20},
		{"negative page clamps to 1", -5, 20, 1, 20},
		{"zero page size defaults to 20", 1, 0, 1, 20},
		{"negative page size defaults to 20", 1, -10, 1, 20},
		{"oversized page size clamps to 100", 1, 500, 1, 100},
		{"page size exactly at cap is preserved", 1, 100, 1, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPage, gotPageSize := clampPagination(tt.page, tt.pageSize)
			if gotPage != tt.wantPage || gotPageSize != tt.wantPageSize {
				t.Errorf("clampPagination(%d, %d) = (%d, %d), want (%d, %d)",
					tt.page, tt.pageSize, gotPage, gotPageSize, tt.wantPage, tt.wantPageSize)
			}
		})
	}
}

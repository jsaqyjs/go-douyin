package profile_service

import "testing"

func TestNormalizePagination(t *testing.T) {
	tests := []struct {
		name      string
		page      int
		limit     int
		wantPage  int
		wantLimit int
	}{
		{
			name:      "正常分页",
			page:      2,
			limit:     10,
			wantPage:  2,
			wantLimit: 10,
		},
		{
			name:      "默认分页",
			page:      0,
			limit:     0,
			wantPage:  1,
			wantLimit: 10,
		},
		{
			name:      "限制最大数量",
			page:      1,
			limit:     1000,
			wantPage:  1,
			wantLimit: 50,
		},
		{
			name:      "限制最大页码",
			page:      999999,
			limit:     20,
			wantPage:  10000,
			wantLimit: 20,
		},
		{
			name:      "负数参数",
			page:      -5,
			limit:     -10,
			wantPage:  1,
			wantLimit: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page, limit := NormalizePagination(
				tt.page,
				tt.limit,
			)

			if page != tt.wantPage {
				t.Errorf(
					"page期望%d，实际%d",
					tt.wantPage,
					page,
				)
			}

			if limit != tt.wantLimit {
				t.Errorf(
					"limit期望%d，实际%d",
					tt.wantLimit,
					limit,
				)
			}
		})
	}
}

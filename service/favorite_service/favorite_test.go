package favorite_service

import "testing"

func TestNormalizeFavoritePagination(t *testing.T) {
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
			name:      "使用默认值",
			page:      0,
			limit:     0,
			wantPage:  1,
			wantLimit: 10,
		},
		{
			name:      "限制最大数量",
			page:      1,
			limit:     100,
			wantPage:  1,
			wantLimit: 50,
		},
		{
			name:      "限制最大页码",
			page:      20000,
			limit:     10,
			wantPage:  10000,
			wantLimit: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page, limit := NormalizeFavoritePagination(
				tt.page,
				tt.limit,
			)

			if page != tt.wantPage ||
				limit != tt.wantLimit {
				t.Fatalf(
					"期望(%d,%d)，实际(%d,%d)",
					tt.wantPage,
					tt.wantLimit,
					page,
					limit,
				)
			}
		})
	}
}

func TestInvalidFavoriteIDs(t *testing.T) {
	// 参数验证发生在数据库访问之前，
	// 所以这个测试不需要连接MySQL。
	err := FavoriteVideo(
		t.Context(), 0, 100,
	)

	if err != ErrInvalidUser {
		t.Fatalf("期望ErrInvalidUser，实际: %v", err)
	}

	err = FavoriteVideo(
		t.Context(), 1, 0,
	)

	if err != ErrInvalidVideo {
		t.Fatalf("期望ErrInvalidVideo，实际: %v", err)
	}
}

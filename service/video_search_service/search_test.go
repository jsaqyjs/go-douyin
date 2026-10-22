package video_search_service

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestNormalizeKeyword(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "正常英文关键词",
			input: " Go ",
			want:  "Go",
		},
		{
			name:  "正常中文关键词",
			input: " 机器学习 ",
			want:  "机器学习",
		},
		{
			name:    "空关键词",
			input:   "   ",
			wantErr: true,
		},
		{
			name:    "关键词过长",
			input:   strings.Repeat("a", 51),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeKeyword(tt.input)

			if tt.wantErr {
				if !errors.Is(err, ErrInvalidKeyword) {
					t.Fatalf("期望关键词校验失败")
				}
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			if got != tt.want {
				t.Fatalf(
					"期望%q，实际%q",
					tt.want, got,
				)
			}
		})
	}
}

func TestEscapeLike(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Go", "Go"},
		{"100%", "100!%"},
		{"a_b", "a!_b"},
		{"a!b", "a!!b"},
	}

	for _, tt := range tests {
		got := EscapeLike(tt.input)

		if got != tt.want {
			t.Errorf(
				"输入%q，期望%q，实际%q",
				tt.input, tt.want, got,
			)
		}
	}
}

func TestSearchCursor(t *testing.T) {
	now := time.Date(
		2026, 10, 22,
		10, 30, 0, 0,
		time.UTC,
	)

	encoded, err := encodeCursor(
		now, 100, "Go",
	)
	if err != nil {
		t.Fatal(err)
	}

	decoded, err := decodeCursor(
		encoded, "Go",
	)
	if err != nil {
		t.Fatal(err)
	}

	if decoded.ID != 100 ||
		decoded.Keyword != "Go" ||
		!decoded.CreatedAt.Equal(now) {
		t.Fatalf("游标解码结果不正确")
	}

	// 将Go的游标拿去搜索Redis，必须拒绝
	_, err = decodeCursor(encoded, "Redis")

	if !errors.Is(err, ErrInvalidCursor) {
		t.Fatal("不同搜索关键词不能混用游标")
	}
}

func TestSearchInvalidCursor(t *testing.T) {
	cases := []string{
		"",
		"invalid!!!",
		"e30",
	}

	for _, raw := range cases {
		_, err := decodeCursor(raw, "Go")
		if !errors.Is(err, ErrInvalidCursor) {
			t.Errorf(
				"无效游标%q没有被拒绝",
				raw,
			)
		}
	}
}

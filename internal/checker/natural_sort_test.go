package checker

import "testing"

func TestNaturalLess(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		// 숫자 앞에 오는 케이스
		{"001.sql", "002.sql", true},
		{"009.sql", "010.sql", true},
		{"9.sql", "10.sql", true}, // numeric: 9 < 10 (lexicographic: "10" < "9")
		{"1.sql", "2.sql", true},
		{"002.sql", "010.sql", true},

		// 숫자 뒤에 오는 케이스
		{"002.sql", "001.sql", false},
		{"010.sql", "009.sql", false},
		{"10.sql", "9.sql", false},

		// 같은 이름
		{"001.sql", "001.sql", false},

		// 접두사 포함 이름
		{"v1_create_users.sql", "v2_add_email.sql", true},
		{"v9_foo.sql", "v10_bar.sql", true}, // numeric: 9 < 10
		{"v10_bar.sql", "v9_foo.sql", false},

		// 확장자 포함
		{"0001_init.sql", "0002_users.sql", true},
		{"0002_users.sql", "0001_init.sql", false},

		// 빈 문자열
		{"", "a", true},
		{"a", "", false},
		{"", "", false},
	}

	for _, tc := range cases {
		got := naturalLess(tc.a, tc.b)
		if got != tc.want {
			t.Errorf("naturalLess(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

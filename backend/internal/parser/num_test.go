package parser

import "testing"

func TestParseFloat(t *testing.T) {
	tests := []struct {
		in      string
		want    float64
		wantErr bool
	}{
		{in: "1,759,000", want: 1759000},   // 콤마 제거
		{in: "122.00", want: 122},          // 소수점 보유수량
		{in: "-3.21%", want: -3.21},        // 손익률: % 제거, 음수 유지
		{in: "-1,234,567", want: -1234567}, // 음수 평가손익
		{in: "0", want: 0},
		{in: " 4,500 ", want: 4500}, // 앞뒤 공백
		{in: "-", wantErr: true},    // 빈 값 표기는 parseFloat 이 거부
		{in: "", wantErr: true},
		{in: "N/A", wantErr: true},
	}

	for _, tt := range tests {
		got, err := parseFloat(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseFloat(%q) = %v, 에러를 기대했음", tt.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseFloat(%q) 예상치 못한 에러: %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("parseFloat(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestParseNullableFloat(t *testing.T) {
	// "-" 와 빈 문자열만 nil 이고, 음수는 값으로 살아있어야 한다.
	for _, in := range []string{"-", "", "  "} {
		got, err := parseNullableFloat(in)
		if err != nil {
			t.Errorf("parseNullableFloat(%q) 예상치 못한 에러: %v", in, err)
		}
		if got != nil {
			t.Errorf("parseNullableFloat(%q) = %v, want nil", in, *got)
		}
	}

	got, err := parseNullableFloat("-123,456")
	if err != nil {
		t.Fatalf("parseNullableFloat(\"-123,456\") 예상치 못한 에러: %v", err)
	}
	if got == nil || *got != -123456 {
		t.Errorf("parseNullableFloat(\"-123,456\") = %v, want -123456", got)
	}

	if _, err := parseNullableFloat("N/A"); err == nil {
		t.Error("parseNullableFloat(\"N/A\") 에러를 기대했음")
	}
}

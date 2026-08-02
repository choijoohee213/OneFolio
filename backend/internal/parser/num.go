package parser

import (
	"fmt"
	"strconv"
	"strings"
)

// parseFloat 은 잔고파일 셀의 숫자 문자열을 float64 로 바꾼다.
// 콤마(1,759,000)와 퍼센트 기호(-3.21%)를 제거한 뒤 변환한다.
func parseFloat(s string) (float64, error) {
	cleaned := strings.NewReplacer(",", "", "%", "", " ", "").Replace(s)
	if cleaned == "" {
		return 0, fmt.Errorf("빈 값은 숫자로 변환할 수 없음: %q", s)
	}
	v, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return 0, fmt.Errorf("숫자 변환 실패: %q", s)
	}
	return v, nil
}

// parseNullableFloat 은 parseFloat 과 같지만, 파일에서 빈 값을 뜻하는 "-" 를
// nil 로 돌려준다. 음수(-123,456)와 혼동하지 않도록 "-" 단독일 때만 nil 이다.
func parseNullableFloat(s string) (*float64, error) {
	if isBlank(s) {
		return nil, nil
	}
	v, err := parseFloat(s)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// isBlank 은 셀이 값 없음을 나타내는지 판별한다. 잔고파일은 빈 값을 "-" 로 쓴다.
func isBlank(s string) bool {
	t := strings.TrimSpace(s)
	return t == "" || t == "-"
}

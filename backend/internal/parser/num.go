package parser

import (
	"fmt"
	"strconv"
	"strings"
)

var cellCleaner = strings.NewReplacer(",", "", "%", "", " ", "")

func parseFloat(s string) (float64, error) {
	cleaned := cellCleaner.Replace(s)
	if cleaned == "" {
		return 0, fmt.Errorf("빈 값은 숫자로 변환할 수 없음: %q", s)
	}
	value, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return 0, fmt.Errorf("숫자 변환 실패: %q", s)
	}
	return value, nil
}

func parseNullableFloat(s string) (*float64, error) {
	if isBlank(s) {
		return nil, nil
	}
	value, err := parseFloat(s)
	if err != nil {
		return nil, err
	}
	return &value, nil
}

// 잔고파일은 빈 값을 "-" 로 쓴다. 음수(-123,456)와 헷갈리지 않게 "-" 단독일 때만이다.
func isBlank(s string) bool {
	trimmed := strings.TrimSpace(s)
	return trimmed == "" || trimmed == "-"
}

func labeled(label string, err error) error {
	return fmt.Errorf("%s %w", label, err)
}

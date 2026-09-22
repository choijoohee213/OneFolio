package ocr

import (
	"errors"
	"testing"
)

var (
	errOverload = errors.New("Error 503, Message: This model is currently experiencing high demand., Status: UNAVAILABLE")
	errTimeout  = errors.New(`doRequest: error sending request: Post "https://...": context deadline exceeded`)
)

// 503 은 모델 단위 과부하라 같은 모델을 다시 부르면 또 503 이 온다.
// 배포 로그에서 2초 간격으로 연달아 503 을 맞은 게 이 때문이었다.
func TestNextAttemptSwitchesModelOnOverload(t *testing.T) {
	c := &Client{model: "primary", fallbackModel: "fallback"}

	model, timeout, ok := c.nextAttempt("primary", errOverload, 0)
	if !ok || model != "fallback" {
		t.Fatalf("과부하면 곧바로 폴백해야 한다: model=%q ok=%v", model, ok)
	}
	if timeout != fallbackTimeout {
		t.Errorf("폴백은 폴백 타임아웃을 써야 한다: %v", timeout)
	}
}

// 타임아웃은 다시 걸면 대개 풀려서 주 모델을 한 번 더 쓴다.
func TestNextAttemptRetriesPrimaryOnFirstTimeout(t *testing.T) {
	c := &Client{model: "primary", fallbackModel: "fallback"}

	if _, _, ok := c.nextAttempt("primary", errTimeout, 0); ok {
		t.Error("첫 타임아웃에는 주 모델을 한 번 더 써야 한다")
	}
	model, _, ok := c.nextAttempt("primary", errTimeout, 1)
	if !ok || model != "fallback" {
		t.Errorf("두 번째 타임아웃이면 마지막 시도는 폴백이어야 한다: model=%q ok=%v", model, ok)
	}
}

func TestNextAttemptKeepsFallbackOnceThere(t *testing.T) {
	c := &Client{model: "primary", fallbackModel: "fallback"}

	if _, _, ok := c.nextAttempt("fallback", errOverload, 1); ok {
		t.Error("이미 폴백이면 더 바꿀 모델이 없다")
	}
}

func TestNextAttemptWithoutFallback(t *testing.T) {
	c := &Client{model: "primary"}

	if _, _, ok := c.nextAttempt("primary", errOverload, 0); ok {
		t.Error("폴백을 꺼 두면 모델을 바꾸지 않는다")
	}
}

func TestRetryableCoversObservedFailures(t *testing.T) {
	for _, err := range []error{errOverload, errTimeout, errors.New("Error 429")} {
		if !retryable(err) {
			t.Errorf("다시 걸어야 할 실패다: %v", err)
		}
	}
	if retryable(errors.New("Error 400, Message: invalid image")) {
		t.Error("400 은 다시 걸어도 같은 결과다")
	}
}

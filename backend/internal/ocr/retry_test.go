package ocr

import (
	"errors"
	"testing"
)

var (
	errOverload = errors.New("Error 503, Message: This model is currently experiencing high demand., Status: UNAVAILABLE")
	errTimeout  = errors.New(`doRequest: error sending request: Post "https://...": context deadline exceeded`)
	// 같은 타임아웃이 서버 쪽에서 먼저 끊겨 돌아온 모습이다.
	errServerTimeout = errors.New("Error 504, Message: Deadline expired before operation could complete., Status: DEADLINE_EXCEEDED, Details: []")
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

// SDK 가 우리 데드라인을 x-server-timeout 으로 서버에 넘겨서, 같은 타임아웃이
// 우리 쪽 context 취소로도 오고 서버의 504 로도 온다. 둘을 다르게 다루면
// 504 로 온 날만 재시도 없이 실패한다.
func TestServerSideTimeoutIsTreatedLikeClientTimeout(t *testing.T) {
	if !retryable(errServerTimeout) {
		t.Fatal("504 는 우리 타임아웃이 서버에서 되돌아온 것이라 다시 걸어야 한다")
	}
	if overloaded(errServerTimeout) {
		t.Error("504 는 과부하가 아니다 — 모델을 바꿀 게 아니라 다시 걸어야 한다")
	}

	c := &Client{model: "primary", fallbackModel: "fallback"}
	if _, _, ok := c.nextAttempt("primary", errServerTimeout, 0); ok {
		t.Error("첫 504 에는 주 모델을 한 번 더 써야 한다")
	}
}

// 크레딧이 0 이 되면 결제계정에 딸린 키가 한꺼번에 멈춘다. 재시도도 모델
// 교체도 소용없으므로 다른 실패와 갈라서 그대로 화면까지 올려야 한다.
func TestCreditExhaustedIsNotRetried(t *testing.T) {
	err := errors.New("Error 402, Message: Payment required, Status: PAYMENT_REQUIRED, Details: []")

	if !creditExhausted(err) {
		t.Fatal("402 는 크레딧 소진으로 가려야 한다")
	}
	if retryable(err) {
		t.Error("크레딧이 없으면 다시 걸어도 같은 결과다")
	}
	if overloaded(err) {
		t.Error("크레딧 소진은 과부하가 아니다 — 모델을 바꿔도 풀리지 않는다")
	}
	if timedOut(err) {
		t.Error("크레딧 소진은 타임아웃이 아니다")
	}
}

func TestCreditExhaustedDoesNotCatchOtherFailures(t *testing.T) {
	for _, err := range []error{errOverload, errTimeout, errServerTimeout} {
		if creditExhausted(err) {
			t.Errorf("크레딧 소진이 아니다: %v", err)
		}
	}
}

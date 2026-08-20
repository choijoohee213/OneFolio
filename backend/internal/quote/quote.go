// Package quote 는 한국투자증권 오픈API로 현재가·환율을 조회한다. 계좌 정보는
// 건드리지 않는다 — 잔고파일/수동입력으로 이미 아는 종목의 현재가만 새로고침하는
// 용도라 시세 조회 엔드포인트만 쓴다.
package quote

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

const (
	baseURL        = "https://openapi.koreainvestment.com:9443"
	requestTimeout = 10 * time.Second
	// 만료 임박 전에 미리 갱신해 요청 도중 만료되는 걸 피한다.
	tokenRefreshMargin = 60 * time.Second
	// ExchangeRate 는 별도 환율 엔드포인트가 없어 해외주식 현재가상세 응답에 같이
	// 오는 당일환율(t_rate)을 빌려 쓴다. 이 값을 이 기간만큼 재사용한다.
	rateCacheTTL = 60 * time.Second
	// 환율만 필요할 때(전용 심볼 없이) 조회에 쓰는 유동성 높은 기준 종목.
	fxAnchorSymbol   = "AAPL"
	fxAnchorExchange = "NAS"
)

// overseasExchanges 는 해외 종목 코드만으로는 거래소를 알 수 없어 순서대로
// 시도해 보는 후보다. OneFolio가 다루는 해외 종목은 대부분 미국 상장이라 이
// 셋으로 충분하다.
var overseasExchanges = []string{"NAS", "NYS", "AMS"}

type Price struct {
	Symbol   string
	Price    float64
	Currency string
	// PrevClose 는 전일 종가다. 현재가를 빨강(상승)·파랑(하락)으로 표시하는
	// 기준이 되며, 값이 없으면 0 이다.
	PrevClose float64
}

type Client struct {
	appKey     string
	appSecret  string
	httpClient *http.Client

	mu          sync.Mutex
	token       string
	tokenExpiry time.Time
	lastRate    float64
	lastRateAt  time.Time
	// exchangeOf 는 해외 종목코드별로 마지막에 맞았던 거래소를 기억해 둔다.
	// 같은 종목을 반복 조회할 때(예: 30초 자동 새로고침) 매번 3곳을 다 시도하지
	// 않고 캐시된 거래소부터 시도한다.
	exchangeOf map[string]string
}

func NewClient(appKey, appSecret string) *Client {
	return &Client{
		appKey:     appKey,
		appSecret:  appSecret,
		httpClient: &http.Client{Timeout: requestTimeout},
		exchangeOf: make(map[string]string),
	}
}

func (c *Client) accessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.token != "" && time.Now().Before(c.tokenExpiry) {
		return c.token, nil
	}

	body, err := json.Marshal(map[string]string{
		"grant_type": "client_credentials",
		"appkey":     c.appKey,
		"appsecret":  c.appSecret,
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/oauth2/tokenP", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("토큰 발급 요청 실패: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("토큰 발급 실패: %s", resp.Status)
	}

	var out struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("토큰 응답 파싱 실패: %w", err)
	}

	c.token = out.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(out.ExpiresIn)*time.Second - tokenRefreshMargin)
	return c.token, nil
}

// envelope 는 한투 API 공통 응답 포맷이다. rt_cd 가 "0"이 아니면 실패다 —
// HTTP 상태코드는 200이어도 실패할 수 있다(예: 잘못된 거래소코드로 조회).
type envelope struct {
	RtCd   string          `json:"rt_cd"`
	Msg1   string          `json:"msg1"`
	Output json.RawMessage `json:"output"`
}

// transientError 는 다시 시도해 볼 만한 실패(네트워크 오류, 5xx)를 표시한다.
// rt_cd 가 "0"이 아닌 논리적 실패(예: 해외 종목 거래소 추측 중 틀린 거래소)는
// transientError 가 아니다 — 재시도해도 결과가 똑같으므로 바로 다음 후보로
// 넘어가는 게 맞다.
type transientError struct{ err error }

func (e *transientError) Error() string { return e.err.Error() }
func (e *transientError) Unwrap() error { return e.err }

const (
	maxRetries = 2
	retryDelay = 300 * time.Millisecond
)

// get 은 transientError 에 한해 짧은 대기 후 재시도한다. 한투 API 가 가끔
// 순간적인 500을 내는 걸 실제로 겪어서(부하 없이도 발생) 넣었다 — 재시도 없이는
// 그 종목만 조용히 빠진 채 응답이 나가서 사용자가 원인을 알 수 없다.
func (c *Client) get(ctx context.Context, path, trID string, params url.Values, out any) error {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(retryDelay)
		}
		err := c.doRequest(ctx, path, trID, params, out)
		if err == nil {
			return nil
		}
		var te *transientError
		if !errors.As(err, &te) {
			return err
		}
		lastErr = err
	}
	return lastErr
}

func (c *Client) doRequest(ctx context.Context, path, trID string, params url.Values, out any) error {
	token, err := c.accessToken(ctx)
	if err != nil {
		return &transientError{err}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+path+"?"+params.Encode(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("authorization", "Bearer "+token)
	req.Header.Set("appkey", c.appKey)
	req.Header.Set("appsecret", c.appSecret)
	req.Header.Set("tr_id", trID)
	req.Header.Set("custtype", "P")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &transientError{fmt.Errorf("한투 API 요청 실패: %w", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return &transientError{fmt.Errorf("한투 API 응답 실패: %s", resp.Status)}
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("한투 API 응답 실패: %s", resp.Status)
	}

	var env envelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return &transientError{fmt.Errorf("한투 API 응답 파싱 실패: %w", err)}
	}
	if env.RtCd != "0" {
		return fmt.Errorf("한투 API 오류: %s", env.Msg1)
	}
	if out != nil {
		if err := json.Unmarshal(env.Output, out); err != nil {
			return fmt.Errorf("한투 API 응답 파싱 실패: %w", err)
		}
	}
	return nil
}

func isDomesticCode(code string) bool {
	if code == "" {
		return false
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func (c *Client) domesticPrice(ctx context.Context, code string) (Price, error) {
	var out struct {
		Prpr string `json:"stck_prpr"`
		Sdpr string `json:"stck_sdpr"`
	}
	params := url.Values{"FID_COND_MRKT_DIV_CODE": {"J"}, "FID_INPUT_ISCD": {code}}
	if err := c.get(ctx, "/uapi/domestic-stock/v1/quotations/inquire-price", "FHKST01010100", params, &out); err != nil {
		return Price{}, err
	}
	price, err := strconv.ParseFloat(out.Prpr, 64)
	if err != nil {
		return Price{}, fmt.Errorf("가격 파싱 실패: %w", err)
	}
	prevClose, _ := strconv.ParseFloat(out.Sdpr, 64)
	return Price{Symbol: code, Price: price, Currency: "KRW", PrevClose: prevClose}, nil
}

// overseasPrice 는 현재가와 함께 당일환율(t_rate)도 돌려준다 — 해외주식
// 현재가상세 응답에 원화환산에 필요한 값이 같이 온다.
func (c *Client) overseasPrice(ctx context.Context, symbol string) (Price, float64, error) {
	c.mu.Lock()
	cached, hasCached := c.exchangeOf[symbol]
	c.mu.Unlock()

	if hasCached {
		if price, rate, err := c.tryOverseasExchange(ctx, symbol, cached); err == nil {
			return price, rate, nil
		}
	}

	var lastErr error
	for _, excd := range overseasExchanges {
		if hasCached && excd == cached {
			continue
		}
		price, rate, err := c.tryOverseasExchange(ctx, symbol, excd)
		if err != nil {
			lastErr = err
			continue
		}
		c.mu.Lock()
		c.exchangeOf[symbol] = excd
		c.mu.Unlock()
		return price, rate, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("%s: 일치하는 거래소를 찾지 못했습니다", symbol)
	}
	return Price{}, 0, lastErr
}

func (c *Client) tryOverseasExchange(ctx context.Context, symbol, excd string) (Price, float64, error) {
	var out struct {
		Last  string `json:"last"`
		Base  string `json:"base"`
		Curr  string `json:"curr"`
		TRate string `json:"t_rate"`
	}
	params := url.Values{"AUTH": {""}, "EXCD": {excd}, "SYMB": {symbol}}
	if err := c.get(ctx, "/uapi/overseas-price/v1/quotations/price-detail", "HHDFS76200200", params, &out); err != nil {
		return Price{}, 0, err
	}
	last, err := strconv.ParseFloat(out.Last, 64)
	if err != nil || last == 0 {
		return Price{}, 0, fmt.Errorf("%s/%s: 시세 없음", excd, symbol)
	}
	rate, _ := strconv.ParseFloat(out.TRate, 64)
	prevClose, _ := strconv.ParseFloat(out.Base, 64)
	currency := out.Curr
	if currency == "" {
		currency = "USD"
	}
	return Price{Symbol: symbol, Price: last, Currency: currency, PrevClose: prevClose}, rate, nil
}

// Prices 는 종목코드(국내는 "005930" 같은 6자리 숫자, 해외는 "AAPL" 같은 티커)로
// 현재가를 조회한다. 국내는 표준시세, 해외는 거래소를 순서대로 시도해 찾는다.
// 개별 종목 조회가 실패해도 나머지는 계속 채운다 — 하나 때문에 전체를 비우지 않는다.
func (c *Client) Prices(ctx context.Context, symbols []string) (map[string]Price, error) {
	result := make(map[string]Price, len(symbols))
	for _, symbol := range symbols {
		if isDomesticCode(symbol) {
			price, err := c.domesticPrice(ctx, symbol)
			if err != nil {
				continue
			}
			result[symbol] = price
			continue
		}

		price, rate, err := c.overseasPrice(ctx, symbol)
		if err != nil {
			continue
		}
		result[symbol] = price
		if rate > 0 {
			c.mu.Lock()
			c.lastRate = rate
			c.lastRateAt = time.Now()
			c.mu.Unlock()
		}
	}
	return result, nil
}

// ExchangeRate 는 baseCurrency 1단위가 quoteCurrency 로 얼마인지 반환한다.
// 한투 API 에는 환율만 단독으로 주는 엔드포인트가 없어, 최근 Prices 호출에서
// 얻은 값을 재사용하거나(rateCacheTTL 이내) 없으면 기준 종목 하나를 조회해서 얻는다.
func (c *Client) ExchangeRate(ctx context.Context, base, quoteCurrency string) (float64, error) {
	if base != "USD" || quoteCurrency != "KRW" {
		return 0, fmt.Errorf("지원하지 않는 통화쌍: %s/%s", base, quoteCurrency)
	}

	c.mu.Lock()
	if c.lastRate > 0 && time.Since(c.lastRateAt) < rateCacheTTL {
		rate := c.lastRate
		c.mu.Unlock()
		return rate, nil
	}
	c.mu.Unlock()

	_, rate, err := c.tryOverseasExchange(ctx, fxAnchorSymbol, fxAnchorExchange)
	if err != nil {
		return 0, err
	}
	if rate == 0 {
		return 0, fmt.Errorf("환율 조회 실패")
	}
	c.mu.Lock()
	c.lastRate = rate
	c.lastRateAt = time.Now()
	c.mu.Unlock()
	return rate, nil
}

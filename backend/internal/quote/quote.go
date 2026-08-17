// Package quote 는 토스증권 Open API로 현재가·환율을 조회한다. 계좌 정보는
// 건드리지 않는다 — 잔고파일/수동입력으로 이미 아는 종목의 현재가만 새로고침하는
// 용도라 시세 조회 엔드포인트(Client Credentials, 계좌 비종속)만 쓴다.
package quote

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	baseURL        = "https://openapi.tossinvest.com"
	requestTimeout = 10 * time.Second
	// 만료 임박 전에 미리 갱신해 요청 도중 만료되는 걸 피한다.
	tokenRefreshMargin = 60 * time.Second
)

type Price struct {
	Symbol   string
	Price    float64
	Currency string
}

type Client struct {
	clientID     string
	clientSecret string
	httpClient   *http.Client

	mu          sync.Mutex
	token       string
	tokenExpiry time.Time
}

func NewClient(clientID, clientSecret string) *Client {
	return &Client{
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient:   &http.Client{Timeout: requestTimeout},
	}
}

func (c *Client) accessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.token != "" && time.Now().Before(c.tokenExpiry) {
		return c.token, nil
	}

	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {c.clientID},
		"client_secret": {c.clientSecret},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/oauth2/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("토큰 발급 요청 실패: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("토큰 발급 실패: %s", resp.Status)
	}

	var body struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("토큰 응답 파싱 실패: %w", err)
	}

	c.token = body.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(body.ExpiresIn)*time.Second - tokenRefreshMargin)
	return c.token, nil
}

func (c *Client) get(ctx context.Context, path string, query url.Values, out any) error {
	token, err := c.accessToken(ctx)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+path+"?"+query.Encode(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("토스 API 요청 실패: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("토스 API 응답 실패: %s", resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("토스 API 응답 파싱 실패: %w", err)
	}
	return nil
}

// Prices 는 종목코드(국내는 "005930" 같은 6자리 숫자, 해외는 "AAPL" 같은 티커)로
// 현재가를 조회한다. 한 번에 최대 200개까지 가능하다.
func (c *Client) Prices(ctx context.Context, symbols []string) (map[string]Price, error) {
	if len(symbols) == 0 {
		return map[string]Price{}, nil
	}

	var body struct {
		Result []struct {
			Symbol    string `json:"symbol"`
			LastPrice string `json:"lastPrice"`
			Currency  string `json:"currency"`
		} `json:"result"`
	}
	query := url.Values{"symbols": {strings.Join(symbols, ",")}}
	if err := c.get(ctx, "/api/v1/prices", query, &body); err != nil {
		return nil, err
	}

	result := make(map[string]Price, len(body.Result))
	for _, item := range body.Result {
		price, err := strconv.ParseFloat(item.LastPrice, 64)
		if err != nil {
			continue
		}
		result[item.Symbol] = Price{Symbol: item.Symbol, Price: price, Currency: item.Currency}
	}
	return result, nil
}

// ExchangeRate 는 baseCurrency 1단위가 quoteCurrency 로 얼마인지 반환한다.
// 토스 Open API 는 KRW, USD 만 지원한다.
func (c *Client) ExchangeRate(ctx context.Context, base, quoteCurrency string) (float64, error) {
	var body struct {
		Result struct {
			Rate string `json:"rate"`
		} `json:"result"`
	}
	query := url.Values{"baseCurrency": {base}, "quoteCurrency": {quoteCurrency}}
	if err := c.get(ctx, "/api/v1/exchange-rate", query, &body); err != nil {
		return 0, err
	}
	rate, err := strconv.ParseFloat(body.Result.Rate, 64)
	if err != nil {
		return 0, fmt.Errorf("환율 파싱 실패: %w", err)
	}
	return rate, nil
}

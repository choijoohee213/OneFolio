package ocr

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"sync/atomic"
	"time"

	"google.golang.org/genai"
)

const requestTimeout = 30 * time.Second

type ExtractedHolding struct {
	Name          string   `json:"name"`
	Ticker        string   `json:"ticker,omitempty"`
	AccountNumber string   `json:"accountNumber,omitempty"`
	AccountType   string   `json:"accountType,omitempty"`
	Quantity      *float64 `json:"quantity"`
	CurrentPrice  *float64 `json:"currentPrice"`
	AvgBuyPrice   *float64 `json:"avgBuyPrice"`
	EvalAmount    *float64 `json:"evalAmount"`
	ProfitLoss    *float64 `json:"profitLoss"`
	ProfitRate    *float64 `json:"profitRate"`
}

type Result struct {
	AccountNumber string             `json:"accountNumber,omitempty"`
	AccountType   string             `json:"accountType,omitempty"`
	Holdings      []ExtractedHolding `json:"holdings"`
}

type accountGroup struct {
	AccountNumber string             `json:"accountNumber"`
	AccountType   string             `json:"accountType"`
	Holdings      []ExtractedHolding `json:"holdings"`
}

type geminiResponse struct {
	Accounts []accountGroup `json:"accounts"`
}

const systemPrompt = `증권 앱 캡처 이미지에서 계좌 정보와 보유 종목을 추출하라.

규칙:
- 계좌번호(예: 111-1111-1111-0)와 계좌유형(예: 종합_주식)이 보이면 추출한다. 안 보이면 빈 문자열로 둔다.
- 여러 계좌가 보이면 accounts 배열에 계좌별로 나눠 넣는다. 계좌 구분이 안 되면 하나의 항목에 모두 넣는다.
- 종목명, 티커(종목코드), 수량, 현재가, 평균매입가, 평가금액, 평가손익, 손익률을 읽는다.
- 종목명이 잘려 있으면(예: "PROSHARES ...") 옆에 보이는 티커(예: TQQQ)를 참고해 정식 명칭을 유추하라. 예: TQQQ → PROSHARES ULTRA PRO QQQ.
- 티커: 해외주식은 알파벳 심볼(예: TQQQ, AMD), 국내주식은 숫자 종목코드(예: 005930)를 넣는다. 안 보이면 빈 문자열.
- 이미지에 보이지 않는 필드는 null로 둔다.
- 숫자는 콤마, 원, %, +, - 기호를 제거한 순수 숫자로 변환한다. 손실(마이너스)이면 음수로.
- 평가손익(profitLoss)과 평가금액(evalAmount)은 반드시 원화(KRW) 기준으로 추출한다. "원화평가손익" 등 원화 값이 있으면 그것을 쓰고, 원화 값이 없으면 null로 둔다.
- 현재가(currentPrice)와 평균매입가(avgBuyPrice)는 화면에 보이는 값을 그대로 추출한다. 달러든 원화든 표시된 숫자를 넣는다.
- 손익률(profitRate)은 퍼센트 숫자를 넣는다.
- 총합계/소계 행은 제외하고 개별 종목만 추출한다.
- 반드시 아래 JSON 형식만 출력하고 다른 텍스트는 쓰지 마라.

출력 형식:
{"accounts":[{"accountNumber":"111-1111-1111-0","accountType":"종합_주식","holdings":[{"name":"종목명","ticker":"TQQQ","quantity":10,"currentPrice":50000,"avgBuyPrice":45000,"evalAmount":500000,"profitLoss":50000,"profitRate":11.11}]}]}`

type Client struct {
	clients []*genai.Client
	model   string
	next    atomic.Uint64
}

func NewClient(apiKeys string) (*Client, error) {
	keys := strings.Split(apiKeys, ",")
	var clients []*genai.Client
	ctx := context.Background()
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		c, err := genai.NewClient(ctx, &genai.ClientConfig{
			APIKey:  k,
			Backend: genai.BackendGeminiAPI,
		})
		if err != nil {
			return nil, fmt.Errorf("gemini 클라이언트 생성 실패: %w", err)
		}
		clients = append(clients, c)
	}
	if len(clients) == 0 {
		return nil, fmt.Errorf("유효한 API 키가 없습니다")
	}
	return &Client{clients: clients, model: "gemini-flash-latest"}, nil
}

func (c *Client) pickClient() *genai.Client {
	idx := c.next.Add(1) - 1
	return c.clients[idx%uint64(len(c.clients))]
}

func (c *Client) KeyCount() int {
	return len(c.clients)
}

// retryable 은 다음 키로 넘어가 볼 만한 실패인지 가린다. 429(한도초과)는
// 키마다 다르게 나는 문제라 재시도할 가치가 있다. 반면 타임아웃은 보통
// 네트워크 경로 자체의 문제라 다른 키로 바꿔도 똑같이 멈춘다 — 재시도하면
// 대기 시간만 키 개수만큼 늘어나 배포 플랫폼의 게이트웨이 타임아웃(502)에
// 걸리기 쉬우므로 즉시 실패시킨다.
func retryable(err error) bool {
	return strings.Contains(err.Error(), "429")
}

func (c *Client) Extract(ctx context.Context, imageData []byte, mimeType string) (*Result, error) {
	contents := []*genai.Content{
		genai.NewContentFromParts([]*genai.Part{
			genai.NewPartFromBytes(imageData, mimeType),
			genai.NewPartFromText("이 증권 앱 캡처에서 보유 종목을 추출해줘."),
		}, genai.RoleUser),
	}

	config := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(systemPrompt, genai.RoleUser),
		Temperature:       genai.Ptr(float32(0.1)),
		ResponseMIMEType:  "application/json",
		// 단순 추출 작업이라 확장 추론(thinking)이 필요 없다. 켜져 있으면
		// 응답 시간이 몇 초~30초+ 로 들쭉날쭉해져서 꺼서 속도를 안정시킨다.
		ThinkingConfig: &genai.ThinkingConfig{ThinkingBudget: genai.Ptr(int32(0))},
	}

	var resp *genai.GenerateContentResponse
	var err error
	tried := 0
	for tried < len(c.clients) {
		client := c.pickClient()
		attemptCtx, cancel := context.WithTimeout(ctx, requestTimeout)
		resp, err = client.Models.GenerateContent(attemptCtx, c.model, contents, config)
		cancel()
		if err == nil {
			break
		}
		if !retryable(err) {
			return nil, fmt.Errorf("gemini 호출 실패: %w", err)
		}
		tried++
	}
	if err != nil {
		return nil, fmt.Errorf("gemini 호출 실패 (모든 키 재시도 소진): %w", err)
	}

	text := resp.Text()

	var parsed geminiResponse
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		return nil, fmt.Errorf("응답 파싱 실패: %w (원문: %s)", err, text)
	}

	var result Result
	for _, acct := range parsed.Accounts {
		for i := range acct.Holdings {
			acct.Holdings[i].AccountNumber = acct.AccountNumber
			acct.Holdings[i].AccountType = acct.AccountType
		}
		fillCalculatedFields(acct.Holdings)
		result.Holdings = append(result.Holdings, acct.Holdings...)
		if result.AccountNumber == "" && acct.AccountNumber != "" {
			result.AccountNumber = acct.AccountNumber
			result.AccountType = acct.AccountType
		}
	}
	return &result, nil
}

func ptr(v float64) *float64 { return &v }

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func fillCalculatedFields(holdings []ExtractedHolding) {
	for i := range holdings {
		h := &holdings[i]

		// evalAmount from profitLoss + profitRate (해외주식도 원화 기준으로 정확)
		if h.EvalAmount == nil && h.ProfitLoss != nil && h.ProfitRate != nil && *h.ProfitRate != 0 {
			buyAmount := *h.ProfitLoss / (*h.ProfitRate / 100)
			h.EvalAmount = ptr(round2(buyAmount + *h.ProfitLoss))
		}

		// evalAmount = currentPrice * quantity (같은 통화일 때)
		if h.EvalAmount == nil && h.CurrentPrice != nil && h.Quantity != nil {
			h.EvalAmount = ptr(round2(*h.CurrentPrice * *h.Quantity))
		}

		// profitLoss = evalAmount - avgBuyPrice * quantity
		if h.ProfitLoss == nil && h.EvalAmount != nil && h.AvgBuyPrice != nil && h.Quantity != nil {
			h.ProfitLoss = ptr(round2(*h.EvalAmount - *h.AvgBuyPrice**h.Quantity))
		}

		// profitRate = profitLoss / buyAmount * 100
		if h.ProfitRate == nil && h.ProfitLoss != nil && h.AvgBuyPrice != nil && h.Quantity != nil {
			buyAmount := *h.AvgBuyPrice * *h.Quantity
			if buyAmount != 0 {
				h.ProfitRate = ptr(round2(*h.ProfitLoss / buyAmount * 100))
			}
		}
	}
}

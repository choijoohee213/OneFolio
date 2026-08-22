package ocr

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/choijoohee213/OneFolio/backend/internal/quote"
	"google.golang.org/genai"
)

const requestTimeout = 30 * time.Second

// 모델은 버전을 박아 고정한다. latest 별칭은 늘 가장 새 모델을 가리키는데,
// 새 모델일수록 트래픽이 몰려 503(과부하)을 자주 뱉는다.
//
// 큰 모델은 시간대를 심하게 타서, 같은 캡처가 2초에 끝나기도 하고 16초가
// 걸리기도 한다. 캡처에서 표를 읽어 옮기는 일에는 lite 로 충분하고 추출
// 결과도 같으면서, 두 시간대 모두 2초 안팎으로 일정했다.
// GEMINI_MODEL 로 바꿀 수 있다.
const defaultModel = "gemini-3.1-flash-lite"

type ExtractedHolding struct {
	Name          string   `json:"name"`
	Ticker        string   `json:"ticker,omitempty"`
	AccountNumber string   `json:"accountNumber,omitempty"`
	AccountType   string   `json:"accountType,omitempty"`
	Quantity      *float64 `json:"quantity"`
	CurrentPrice  *float64 `json:"currentPrice"`
	AvgBuyPrice   *float64 `json:"avgBuyPrice"`
	// Currency 는 currentPrice·avgBuyPrice 가 찍힌 통화다("KRW" 또는 "USD").
	// evalAmount·profitLoss·profitRate 는 항상 원화라 여기 포함되지 않는다.
	Currency   string   `json:"currency,omitempty"`
	EvalAmount *float64 `json:"evalAmount"`
	ProfitLoss *float64 `json:"profitLoss"`
	ProfitRate *float64 `json:"profitRate"`
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
- currency: currentPrice·avgBuyPrice 가 찍힌 통화. 화면에 "$"나 "USD"가 보이거나 해외 종목인데 숫자가 작으면(예: 주당 70) 달러일 가능성이 높다 — 그럴 땐 "USD". 원화로 보이면 "KRW". 확실하지 않으면 "KRW"로 둔다.
- 손익률(profitRate)은 퍼센트 숫자를 넣는다.
- 총합계/소계 행은 제외하고 개별 종목만 추출한다.
- 반드시 아래 JSON 형식만 출력하고 다른 텍스트는 쓰지 마라.

출력 형식:
{"accounts":[{"accountNumber":"111-1111-1111-0","accountType":"종합_주식","holdings":[{"name":"종목명","ticker":"TQQQ","quantity":10,"currentPrice":70.5,"avgBuyPrice":65.2,"currency":"USD","evalAmount":500000,"profitLoss":50000,"profitRate":11.11}]}]}`

type Client struct {
	clients []*genai.Client
	model   string
	next    atomic.Uint64
	// quoteClient 는 해외 종목의 평단가·현재가가 달러로 찍혀 있을 때 원화
	// 평가손익을 정확히 환산하는 데 쓴다. nil 이면 환산 없이 null로 둔다.
	quoteClient *quote.Client
}

func NewClient(apiKeys string, quoteClient *quote.Client) (*Client, error) {
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
	model := os.Getenv("GEMINI_MODEL")
	if model == "" {
		model = defaultModel
	}
	return &Client{clients: clients, model: model, quoteClient: quoteClient}, nil
}

func (c *Client) pickClient() *genai.Client {
	idx := c.next.Add(1) - 1
	return c.clients[idx%uint64(len(c.clients))]
}

func (c *Client) KeyCount() int {
	return len(c.clients)
}

// retryable 은 다시 걸어볼 만한 실패인지 가린다. 429(한도초과)는 키마다 다르게
// 나므로 다음 키로 넘어가면 풀린다. 503(과부하)은 키와 무관하지만 잠깐 뒤에는
// 풀리기도 해서 한 번은 더 걸어본다. 타임아웃은 보통 네트워크 경로 자체의
// 문제라 다시 걸어도 똑같이 멈춘다 — 대기 시간만 키 개수만큼 늘어나 배포
// 플랫폼의 게이트웨이 타임아웃에 걸리기 쉬우므로 즉시 실패시킨다.
func retryable(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "429") || strings.Contains(msg, "503") || strings.Contains(msg, "UNAVAILABLE")
}

// 503 은 즉시 오지 않고 수십 초 걸려 오기도 한다. 키 개수만큼 다 돌면 그 시간이
// 그대로 쌓이므로 시도 횟수를 따로 묶는다.
const maxAttempts = 2

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

	attempts := maxAttempts
	if len(c.clients) < attempts {
		attempts = len(c.clients)
	}

	var resp *genai.GenerateContentResponse
	var err error
	for tried := 0; tried < attempts; tried++ {
		client := c.pickClient()
		attemptCtx, cancel := context.WithTimeout(ctx, requestTimeout)
		start := time.Now()
		resp, err = client.Models.GenerateContent(attemptCtx, c.model, contents, config)
		cancel()
		if err == nil {
			break
		}
		// 실패는 화면에 이유가 다 드러나지 않아 로그로 남긴다.
		log.Printf("OCR 실패 (%s, %d/%d, %.1fs): %v", c.model, tried+1, attempts, time.Since(start).Seconds(), err)
		if !retryable(err) {
			return nil, fmt.Errorf("gemini 호출 실패: %w", err)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("gemini 호출 실패 (%d회 시도): %w", attempts, err)
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
		result.Holdings = append(result.Holdings, acct.Holdings...)
		if result.AccountNumber == "" && acct.AccountNumber != "" {
			result.AccountNumber = acct.AccountNumber
			result.AccountType = acct.AccountType
		}
	}

	fillCalculatedFields(result.Holdings, c.usdKrwRate(ctx, result.Holdings))
	return &result, nil
}

// usdKrwRate 는 달러로 찍힌 종목이 하나라도 있을 때만 환율을 조회한다.
// 조회에 실패하거나 시세 클라이언트가 없으면 nil — fillCalculatedFields 는
// 그 경우 원화 계산이 필요한 필드를 채우지 않고 비워 둔다.
func (c *Client) usdKrwRate(ctx context.Context, holdings []ExtractedHolding) *float64 {
	if c.quoteClient == nil {
		return nil
	}
	needsFx := false
	for _, h := range holdings {
		if h.Currency == "USD" {
			needsFx = true
			break
		}
	}
	if !needsFx {
		return nil
	}
	rate, err := c.quoteClient.ExchangeRate(ctx, "USD", "KRW")
	if err != nil {
		return nil
	}
	return &rate
}

// RecomputeKRW 는 currentPrice·avgBuyPrice 를 usdKrw 로 원화 환산해서 evalAmount·
// profitLoss·profitRate 를 무조건 다시 낸다 — 이미 값이 있어도 덮어쓴다.
// 종목마스터로 해외 종목임을 확정했을 때만 불러야 한다: Gemini 가 화면에서
// 직접 읽은 evalAmount·profitLoss 는 통화를 혼동했을 수 있어 currentPrice·
// avgBuyPrice(숫자만 베끼면 되는 값)에서 다시 계산한 값을 더 신뢰할 수 있다.
func RecomputeKRW(h *ExtractedHolding, usdKrw float64) {
	if h.CurrentPrice != nil && h.Quantity != nil {
		h.EvalAmount = ptr(round2(*h.CurrentPrice * usdKrw * *h.Quantity))
	}
	if h.EvalAmount != nil && h.AvgBuyPrice != nil && h.Quantity != nil {
		buyAmount := *h.AvgBuyPrice * usdKrw * *h.Quantity
		profit := *h.EvalAmount - buyAmount
		h.ProfitLoss = ptr(round2(profit))
		if buyAmount != 0 {
			h.ProfitRate = ptr(round2(profit / buyAmount * 100))
		} else {
			h.ProfitRate = nil
		}
	}
}

func ptr(v float64) *float64 { return &v }

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// fillCalculatedFields 는 Gemini 가 비워 둔 파생 필드를 채운다. currentPrice·
// avgBuyPrice 가 달러(Currency=="USD")면 usdKrw 로 원화 환산한 뒤 계산한다.
// 환율을 모르면(usdKrw==nil) 그 종목의 원화 계산은 건너뛴다 — 잘못된 값을
// 만드느니 비워 두는 편이 낫다.
func fillCalculatedFields(holdings []ExtractedHolding, usdKrw *float64) {
	for i := range holdings {
		h := &holdings[i]

		fx := 1.0
		fxKnown := true
		if h.Currency == "USD" {
			if usdKrw == nil {
				fxKnown = false
			} else {
				fx = *usdKrw
			}
		}

		// evalAmount from profitLoss + profitRate — 이미 원화 값이라 환율이 필요 없다.
		if h.EvalAmount == nil && h.ProfitLoss != nil && h.ProfitRate != nil && *h.ProfitRate != 0 {
			buyAmount := *h.ProfitLoss / (*h.ProfitRate / 100)
			h.EvalAmount = ptr(round2(buyAmount + *h.ProfitLoss))
		}

		// evalAmount = currentPrice(원화 환산) * quantity
		if h.EvalAmount == nil && h.CurrentPrice != nil && h.Quantity != nil && fxKnown {
			h.EvalAmount = ptr(round2(*h.CurrentPrice * fx * *h.Quantity))
		}

		// profitLoss = evalAmount - avgBuyPrice(원화 환산) * quantity
		if h.ProfitLoss == nil && h.EvalAmount != nil && h.AvgBuyPrice != nil && h.Quantity != nil && fxKnown {
			h.ProfitLoss = ptr(round2(*h.EvalAmount - *h.AvgBuyPrice*fx**h.Quantity))
		}

		// profitRate = profitLoss / buyAmount * 100
		if h.ProfitRate == nil && h.ProfitLoss != nil && h.AvgBuyPrice != nil && h.Quantity != nil && fxKnown {
			buyAmount := *h.AvgBuyPrice * fx * *h.Quantity
			if buyAmount != 0 {
				h.ProfitRate = ptr(round2(*h.ProfitLoss / buyAmount * 100))
			}
		}
	}
}

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

// 정상 응답은 2~4초에 온다. 한참 기다리다 실패하느니 일찍 끊고 다시 거는 편이
// 빠르다 — 배포 환경에서 관측된 지연은 다시 걸면 대개 곧바로 풀린다.
const requestTimeout = 15 * time.Second

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
	// BuyAmount 는 화면에 매입금액이 보일 때만 채운다. 평단×수량으로 내면
	// 화면의 평단이 반올림된 값이라 원 단위가 어긋난다.
	BuyAmount  *float64 `json:"buyAmount"`
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
- 종목명, 티커(종목코드), 수량, 현재가, 평균매입가, 매입금액, 평가금액, 평가손익, 손익률을 읽는다.
- 매입금액(buyAmount)은 화면에 그 값이 보일 때만 넣는다. 안 보이면 null 로 둔다 — 평균매입가에 수량을 곱해 지어내지 마라.
- 종목명은 화면에 적힌 그대로 옮긴다. 번역하거나 영문·정식 명칭으로 바꾸지 마라 —
  "SK하이닉스(ADR)" 은 "SK Hynix" 가 아니라 "SK하이닉스(ADR)" 이다.
- 종목명이 두 줄로 나뉘어 있으면 이어 붙인다. 예: "TSMC(AD" + "R)" → "TSMC(ADR)".
- 말줄임표로 진짜 잘려 있을 때만(예: "PROSHARES ...") 옆에 보이는 티커를 참고해 뒷부분을 채운다.
- 티커: 화면에 보일 때만 넣는다. 해외주식은 알파벳 심볼(예: TQQQ, AMD), 국내주식은
  숫자 종목코드(예: 005930). 화면에 없으면 빈 문자열로 두고 짐작해서 지어내지 마라.
- 이미지에 보이지 않는 필드는 null로 둔다.
- 숫자는 콤마, 원, %, +, - 기호를 제거한 순수 숫자로 변환한다. 손실(마이너스)이면 음수로.
- 평가손익(profitLoss)과 평가금액(evalAmount)은 반드시 원화(KRW) 기준으로 추출한다. "원화평가손익" 등 원화 값이 있으면 그것을 쓰고, 원화 값이 없으면 null로 둔다.
- 현재가(currentPrice)와 평균매입가(avgBuyPrice)는 화면에 보이는 값을 그대로 추출한다. 달러든 원화든 표시된 숫자를 넣는다.
- currency: currentPrice·avgBuyPrice 가 찍힌 통화. 화면에 적힌 표시로만 정한다 — "$"나 "USD" 기호, "원" 단위, "외/원" 같은 통화 토글에서 켜진 쪽, 컬럼 제목("원화평가손익" 등). 숫자 크기로 짐작하지 마라. 표시가 없으면 "KRW"로 둔다.
- 손익률(profitRate)은 퍼센트 숫자를 넣는다.
- 총합계/소계 행은 제외하고 개별 종목만 추출한다.
- 반드시 아래 JSON 형식만 출력하고 다른 텍스트는 쓰지 마라.

출력 형식:
{"accounts":[{"accountNumber":"111-1111-1111-0","accountType":"종합_주식","holdings":[{"name":"종목명","ticker":"TQQQ","quantity":10,"currentPrice":70.5,"avgBuyPrice":65.2,"currency":"USD","evalAmount":500000,"buyAmount":450000,"profitLoss":50000,"profitRate":11.11}]}]}`

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
// 풀리기도 해서 한 번은 더 걸어본다.
//
// 타임아웃도 다시 건다. 전에는 경로 자체의 문제라 보고 즉시 실패시켰는데,
// 배포 환경에서 재어 보니 그렇지 않았다 — 다섯 번에 한 번쯤 30초를 넘기지만
// 곧바로 다시 걸면 2~3초에 돌아온다. 규칙성이 없어 미리 피할 수도 없다.
func retryable(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "429") ||
		strings.Contains(msg, "503") ||
		strings.Contains(msg, "UNAVAILABLE") ||
		strings.Contains(msg, "context deadline exceeded")
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

	// 파생 필드는 여기서 채우지 않는다. 달러인지 원화인지는 종목마스터로
	// 확정한 뒤라야 알 수 있어서, 호출한 쪽이 FillCalculated 로 마무리한다.
	return &result, nil
}

// usdKrwRate 는 달러로 찍힌 종목이 하나라도 있을 때만 환율을 조회한다.
// 조회에 실패하거나 시세 클라이언트가 없으면 nil — fillCalculatedFields 는
// 그 경우 원화 계산이 필요한 필드를 채우지 않고 비워 둔다.
func (c *Client) UsdKrwRate(ctx context.Context, holdings []ExtractedHolding) *float64 {
	if c.quoteClient == nil {
		return nil
	}
	needsFx := false
	for _, h := range holdings {
		if h.Currency == usd {
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

func ptr(v float64) *float64 { return &v }

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// FillCalculated 는 화면에서 읽지 못한 값만 계산해서 채운다. 화면에 있던 값은
// 절대 건드리지 않는다 — 증권사가 계산해 둔 값이 우리 환율로 다시 낸 값보다
// 믿을 만하다.
//
// currentPrice·avgBuyPrice 가 달러(Currency=="USD")면 usdKrw 로 원화 환산한 뒤
// 계산한다. 환율을 모르면(usdKrw==nil) 그 종목의 원화 계산은 건너뛴다 —
// 잘못된 값을 만드느니 비워 두는 편이 낫다.
func FillCalculated(holdings []ExtractedHolding, usdKrw *float64) {
	for i := range holdings {
		h := &holdings[i]

		fx := 1.0
		fxKnown := true
		if h.Currency == usd {
			if usdKrw == nil {
				fxKnown = false
			} else {
				fx = *usdKrw
			}
		}

		if h.EvalAmount == nil {
			// 원화로 찍힌 화면은 현재가 × 수량이 평가금액과 딱 맞는다. 손익률에서
			// 역산하면 화면의 손익률이 소수점 둘째 자리까지뿐이라 몇 천 원씩 어긋난다.
			if h.Currency != usd && h.CurrentPrice != nil && h.Quantity != nil {
				h.EvalAmount = ptr(round2(*h.CurrentPrice * *h.Quantity))
			} else if v, ok := krwEvalAmount(*h); ok {
				// 달러로 찍힌 화면은 평가손익·손익률이 이미 원화라, 여기서 역산하면
				// 증권사가 쓴 환율이 그대로 실린다. 우리 환율로 곱하는 것보다 낫다.
				h.EvalAmount = ptr(round2(v))
			} else if h.CurrentPrice != nil && h.Quantity != nil && fxKnown {
				h.EvalAmount = ptr(round2(*h.CurrentPrice * fx * *h.Quantity))
			}
		}

		// 매입금액은 화면에서 읽은 값을 우선한다. 평단은 반올림돼 있어서
		// 수량을 곱하면 원 단위가 어긋난다(예: 271,222×9=2,440,998, 실제 2,441,000).
		buyAmount := func() *float64 {
			if h.BuyAmount != nil {
				return h.BuyAmount
			}
			if h.AvgBuyPrice != nil && h.Quantity != nil && fxKnown {
				return ptr(*h.AvgBuyPrice * fx * *h.Quantity)
			}
			return nil
		}()

		// profitLoss = evalAmount - 매입금액
		if h.ProfitLoss == nil && h.EvalAmount != nil && buyAmount != nil {
			h.ProfitLoss = ptr(round2(*h.EvalAmount - *buyAmount))
		}

		// profitRate = profitLoss / 매입금액 * 100
		if h.ProfitRate == nil && h.ProfitLoss != nil && buyAmount != nil && *buyAmount != 0 {
			h.ProfitRate = ptr(round2(*h.ProfitLoss / *buyAmount * 100))
		}
	}
}

const (
	krw = "KRW"
	usd = "USD"

	// 환율로 볼 수 있는 범위. 원/달러가 크게 움직여도 이 밖으로 나가지는 않는다.
	minPlausibleFx = 200.0
	maxPlausibleFx = 5000.0
)

// krwEvalAmount 는 평가금액을 원화로 돌려준다. 화면에 있으면 그 값이고,
// 없으면 평가손익과 손익률로 역산한다 — 둘 다 원화 값이라 환율이 필요 없다.
func krwEvalAmount(h ExtractedHolding) (float64, bool) {
	if h.EvalAmount != nil {
		return *h.EvalAmount, true
	}
	if h.ProfitLoss != nil && h.ProfitRate != nil && *h.ProfitRate != 0 {
		buyAmount := *h.ProfitLoss / (*h.ProfitRate / 100)
		return buyAmount + *h.ProfitLoss, true
	}
	return 0, false
}

// ResolveCurrency 는 화면에 찍힌 현재가·평균매입가가 어느 통화인지 정한다.
//
// 종목이 해외냐로 정하면 안 된다 — 해외 종목이라도 증권사 화면을 원화 보기로
// 두면 원화가 찍힌다. 그걸 달러로 오인하면 값이 환율배만큼 부풀어 오른다.
//
// 캡처 안의 값끼리 비교해 가린다. 평가금액은 늘 원화라(원화 값만 읽게 되어
// 있다) 주당 평가금액을 현재가로 나눈 값이 1 근처면 현재가도 원화고, 환율
// 규모면 달러다. 비교할 값이 없을 때만 모델이 읽은 통화 표시를 믿는다.
// ResolveCurrencies 는 캡처 한 장의 종목들 통화를 한꺼번에 정한다.
//
// 통화는 줄이 아니라 화면(열)의 성질이다. 그래서 값으로 확실히 가려진 줄이
// 하나라도 있으면, 못 가린 줄도 같은 통화로 본다 — 손익이 정확히 0 인 종목은
// 손익률도 0 이라 나눠 볼 수가 없는데, 그 한 줄 때문에 통화를 놓치면 값이
// 환율배만큼 어긋난다.
//
// domestic 은 종목마스터가 국내로 확정한 종목이다. 국내 종목은 늘 원화라
// 화면 통화를 물려받지 않는다.
func ResolveCurrencies(holdings []ExtractedHolding, domestic []bool) {
	screen := ""
	for i, h := range holdings {
		if i < len(domestic) && domestic[i] {
			continue
		}
		if _, ok := pricePerShareRatio(h); ok {
			screen = ResolveCurrency(h, false)
			if screen == usd {
				break
			}
		}
	}
	for i := range holdings {
		isDomestic := i < len(domestic) && domestic[i]
		if _, ok := pricePerShareRatio(holdings[i]); !ok && screen != "" && !isDomestic {
			holdings[i].Currency = screen
			continue
		}
		holdings[i].Currency = ResolveCurrency(holdings[i], isDomestic)
	}
}

func ResolveCurrency(h ExtractedHolding, domesticListing bool) string {
	if domesticListing {
		return krw
	}
	if ratio, ok := pricePerShareRatio(h); ok {
		if ratio >= minPlausibleFx && ratio <= maxPlausibleFx {
			return usd
		}
		return krw
	}
	if h.Currency == usd {
		return usd
	}
	return krw
}

func pricePerShareRatio(h ExtractedHolding) (float64, bool) {
	eval, ok := krwEvalAmount(h)
	if !ok || h.Quantity == nil || *h.Quantity == 0 || h.CurrentPrice == nil || *h.CurrentPrice == 0 {
		return 0, false
	}
	return eval / *h.Quantity / *h.CurrentPrice, true
}

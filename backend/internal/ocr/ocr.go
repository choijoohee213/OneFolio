package ocr

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"google.golang.org/genai"
)

type ExtractedHolding struct {
	Name         string   `json:"name"`
	Ticker       string   `json:"ticker,omitempty"`
	Quantity     *float64 `json:"quantity"`
	CurrentPrice *float64 `json:"currentPrice"`
	AvgBuyPrice  *float64 `json:"avgBuyPrice"`
	EvalAmount   *float64 `json:"evalAmount"`
	ProfitLoss   *float64 `json:"profitLoss"`
	ProfitRate   *float64 `json:"profitRate"`
}

type Result struct {
	AccountNumber string             `json:"accountNumber,omitempty"`
	AccountType   string             `json:"accountType,omitempty"`
	Holdings      []ExtractedHolding `json:"holdings"`
}

const systemPrompt = `증권 앱 캡처 이미지에서 계좌 정보와 보유 종목을 추출하라.

규칙:
- 이미지 상단에 계좌번호(예: 616-8228-7204-0)와 계좌유형(예: 종합_주식)이 보이면 추출한다. 안 보이면 빈 문자열로 둔다.
- 종목명, 티커(종목코드), 수량, 현재가, 평균매입가, 평가금액, 평가손익, 손익률을 읽는다.
- 종목명이 잘려 있으면(예: "PROSHARES ...") 옆에 보이는 티커(예: TQQQ)를 참고해 정식 명칭을 유추하라. 예: TQQQ → PROSHARES ULTRA PRO QQQ.
- 티커가 보이면 ticker 필드에 넣는다. 안 보이면 빈 문자열로 둔다.
- 이미지에 보이지 않는 필드는 null로 둔다.
- 숫자는 콤마, 원, %, +, - 기호를 제거한 순수 숫자로 변환한다. 손실(마이너스)이면 음수로.
- 금액은 반드시 원화(KRW) 기준으로 추출한다. 해외주식이라도 이미지에 원화 환산 금액이 보이면 그 값을 쓴다. 달러 등 외화로만 표시된 금액은 null로 둔다.
- 총합계/소계 행은 제외하고 개별 종목만 추출한다.
- 여러 계좌가 보이면 모두 추출한다.
- 반드시 아래 JSON 형식만 출력하고 다른 텍스트는 쓰지 마라.

출력 형식:
{"accountNumber":"616-8228-7204-0","accountType":"종합_주식","holdings":[{"name":"종목명","ticker":"TQQQ","quantity":10,"currentPrice":50000,"avgBuyPrice":45000,"evalAmount":500000,"profitLoss":50000,"profitRate":11.11}]}`

type Client struct {
	client *genai.Client
	model  string
}

func NewClient(apiKey string) (*Client, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("gemini 클라이언트 생성 실패: %w", err)
	}
	return &Client{client: client, model: "gemini-3.6-flash"}, nil
}

func (c *Client) Extract(ctx context.Context, imageData []byte, mimeType string) (*Result, error) {
	contents := []*genai.Content{
		genai.NewContentFromParts([]*genai.Part{
			genai.NewPartFromBytes(imageData, mimeType),
			genai.NewPartFromText("이 증권 앱 캡처에서 보유 종목을 추출해줘."),
		}, genai.RoleUser),
	}

	resp, err := c.client.Models.GenerateContent(ctx, c.model, contents, &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(systemPrompt, genai.RoleUser),
		Temperature:       genai.Ptr(float32(0.1)),
		ResponseMIMEType:  "application/json",
	})
	if err != nil {
		return nil, fmt.Errorf("gemini 호출 실패: %w", err)
	}

	text := resp.Text()

	var result Result
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, fmt.Errorf("응답 파싱 실패: %w (원문: %s)", err, text)
	}
	fillCalculatedFields(result.Holdings)
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

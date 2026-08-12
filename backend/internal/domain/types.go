package domain

// NormalizeAccountNumber 는 숫자만 남긴다. 같은 계좌라도 잔고파일의 전체 계좌현황은
// 4-4-4-1, 상품보유현황 헤더는 3-2-7 로 하이픈 위치가 달라서 그대로는 join 이 안 된다.
// 사용자가 직접 입력한 계좌번호를 파일 계좌와 맞춰볼 때도 같은 규칙을 쓴다.
func NormalizeAccountNumber(s string) string {
	digits := make([]rune, 0, len(s))
	for _, r := range s {
		if r >= '0' && r <= '9' {
			digits = append(digits, r)
		}
	}
	return string(digits)
}

type Account struct {
	// Number 는 하이픈을 제거한 계좌번호다. 파일 안에서도 하이픈 위치가 갈려서
	// 정규화해야 Holding 과 이어붙는다.
	Number        string  `json:"number"`
	DisplayNumber string  `json:"displayNumber"`
	Type          string  `json:"type"`
	TotalAsset    float64 `json:"totalAsset"`
	Withdrawable  float64 `json:"withdrawable"`
}

// 현금성 자산은 평단·손익이 파일에 "-" 로 비어 있어 포인터로 둔다.
type Holding struct {
	AccountNumber string   `json:"accountNumber"`
	Name          string   `json:"name"`
	Quantity      float64  `json:"quantity"`
	CurrentPrice  *float64 `json:"currentPrice"`
	AvgBuyPrice   *float64 `json:"avgBuyPrice"`
	BuyAmount     *float64 `json:"buyAmount"`
	EvalAmount    float64  `json:"evalAmount"`
	ProfitLoss    *float64 `json:"profitLoss"`
	ProfitRate    *float64 `json:"profitRate"`
}

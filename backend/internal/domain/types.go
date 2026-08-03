package domain

type Account struct {
	// Number 는 하이픈을 제거한 계좌번호다. 파일 안에서도 하이픈 위치가 갈려서
	// 정규화해야 Holding 과 이어붙는다.
	Number        string
	DisplayNumber string
	Type          string
	TotalAsset    float64
	Withdrawable  float64
}

// 현금성 자산은 평단·손익이 파일에 "-" 로 비어 있어 포인터로 둔다.
type Holding struct {
	AccountNumber string
	Name          string
	Quantity      float64
	CurrentPrice  *float64
	AvgBuyPrice   *float64
	BuyAmount     *float64
	EvalAmount    float64
	ProfitLoss    *float64
	ProfitRate    *float64
}

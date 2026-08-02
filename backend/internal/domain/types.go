// Package domain 은 잔고파일에서 읽어낸 자산 데이터의 공용 타입을 정의한다.
package domain

// Account 는 잔고파일 "전체 계좌현황" 테이블의 한 행이다.
type Account struct {
	// Number 는 하이픈을 제거한 계좌번호다. 같은 계좌라도 전체 계좌현황은
	// "616-8228-7204-0", 상품보유현황 헤더는 "616-82-2872040" 으로 표기가
	// 달라서, Holding 과 이어붙이려면 정규화된 값이 필요하다.
	Number        string
	DisplayNumber string  // 파일에 표시된 형식 그대로 (화면 표시용)
	Type          string  // 계좌유형 (ISA(중개형) / 종합 / 연금저축계좌(신))
	TotalAsset    float64 // 자산총액
	Withdrawable  float64 // 출금가능액
}

// Holding 은 잔고파일 "[계좌번호] 상품보유현황" 테이블의 한 행이다.
//
// 현금성 자산(미국달러, 예수금 등)은 평단·손익 관련 값이 파일에 "-" 로 비어
// 있으므로 해당 필드는 포인터로 두고 nil 을 허용한다.
type Holding struct {
	AccountNumber string   // 소속 계좌 (Account.Number 와 같은 정규화된 형식)
	Name          string   // 상품명
	Quantity      float64  // 보유수량 (소수점 거래가 있어 float)
	CurrentPrice  *float64 // 현재가
	AvgBuyPrice   *float64 // 평균매입가
	BuyAmount     *float64 // 매입금액
	EvalAmount    float64  // 평가금액
	ProfitLoss    *float64 // 평가손익
	ProfitRate    *float64 // 손익률 (%, 예: -3.21)
}

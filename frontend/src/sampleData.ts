const ACCOUNT_HEADER = `<html>
<head><meta http-equiv="Content-Type" content="text/html; charset=utf-8"></head>
<body>
<table><tr><td height="30" colspan="8">계좌별잔고</td></tr></table>
<table><tr><td></td></tr></table>
<table><tr><td>전체 계좌현황</td><td colspan="4">(단위: 원)</td></tr></table>
<table class="col_type" summary="계좌번호, 계좌유형, 계좌별명, 자산총액, 출금가능액, 바로가기로 구성된 표 입니다.">
<tr><th>계좌번호</th><th>계좌유형</th><th>계좌별명</th><th>자산총액</th><th>출금가능액</th><th>바로가기</th></tr>
<tr><td>111-1111-1111-0</td><td>ISA(중개형)</td><td></td><td>12,500,000</td><td>500,000</td><td>계좌상세\t\t이체</td></tr>
<tr><td>222-2222-2222-0</td><td>종합</td><td></td><td>25,000,000</td><td>1,200,000</td><td>계좌상세\t\t이체</td></tr>
<tr><td>333-3333-3333-0</td><td>연금저축계좌(신)</td><td></td><td>8,000,000</td><td>0</td><td>계좌상세\t\t이체</td></tr>
</table>
<table><tr><td></td></tr></table>`

const HOLDINGS_FOOTER = `</table>
</body></html>`

const ISA_HOLDINGS = `<table><tr><td colspan="7">[111-11-1111110] 상품보유현황</td></tr></table>
<table class="col_type" summary="상품명, 보유수량, 현재가, 평균매입가, 매입금액, 평가금액, 평가손익, 손익률로 구성된 표 입니다.">
<tr><th>상품명</th><th>보유수량</th><th>현재가</th><th>평균매입가</th><th>매입금액</th><th>평가금액</th><th>평가손익</th><th>손익률</th></tr>
<tr><td>TIGER 미국S&amp;P500</td><td>200.00</td><td>18,500</td><td>16,000</td><td>3,200,000</td><td>3,700,000</td><td>500,000</td><td>15.63%</td></tr>
<tr><td>TIGER 미국나스닥100</td><td>80.00</td><td>105,000</td><td>95,000</td><td>7,600,000</td><td>8,400,000</td><td>800,000</td><td>10.53%</td></tr>`

const JONGHAP_HOLDINGS = `<table><tr><td colspan="7">[222-22-2222220] 상품보유현황</td></tr></table>
<table class="col_type" summary="상품명, 보유수량, 현재가, 평균매입가, 매입금액, 평가금액, 평가손익, 손익률로 구성된 표 입니다.">
<tr><th>상품명</th><th>보유수량</th><th>현재가</th><th>평균매입가</th><th>매입금액</th><th>평가금액</th><th>평가손익</th><th>손익률</th></tr>
<tr><td>삼성전자</td><td>50.00</td><td>72,000</td><td>65,000</td><td>3,250,000</td><td>3,600,000</td><td>350,000</td><td>10.77%</td></tr>
<tr><td>SK하이닉스</td><td>15.00</td><td>220,000</td><td>180,000</td><td>2,700,000</td><td>3,300,000</td><td>600,000</td><td>22.22%</td></tr>
<tr><td>KODEX 200</td><td>100.00</td><td>35,000</td><td>32,000</td><td>3,200,000</td><td>3,500,000</td><td>300,000</td><td>9.38%</td></tr>
<tr><td>애플</td><td>10.00</td><td>310,000</td><td>280,000</td><td>2,800,000</td><td>3,100,000</td><td>300,000</td><td>10.71%</td></tr>
<tr><td>PROSHARES ULTRAPRO QQQ ETF</td><td>30.00</td><td>120,000</td><td>95,000</td><td>2,850,000</td><td>3,600,000</td><td>750,000</td><td>26.32%</td></tr>
<tr><td>미국달러</td><td>500.00</td><td>1,380</td><td>-</td><td>690,000</td><td>690,000</td><td>-</td><td>-</td></tr>`

const PENSION_HOLDINGS = `<table><tr><td colspan="7">[333-33-3333330] 상품보유현황</td></tr></table>
<table class="col_type" summary="상품명, 보유수량, 현재가, 평균매입가, 매입금액, 평가금액, 평가손익, 손익률로 구성된 표 입니다.">
<tr><th>상품명</th><th>보유수량</th><th>현재가</th><th>평균매입가</th><th>매입금액</th><th>평가금액</th><th>평가손익</th><th>손익률</th></tr>
<tr><td>TIGER 미국S&amp;P500</td><td>150.00</td><td>18,500</td><td>17,000</td><td>2,550,000</td><td>2,775,000</td><td>225,000</td><td>8.82%</td></tr>
<tr><td>ACE 미국30년국채액티브(H)</td><td>300.00</td><td>10,500</td><td>11,000</td><td>3,300,000</td><td>3,150,000</td><td>-150,000</td><td>-4.55%</td></tr>
<tr><td>KODEX 미국S&amp;P500TR</td><td>120.00</td><td>15,800</td><td>14,500</td><td>1,740,000</td><td>1,896,000</td><td>156,000</td><td>8.97%</td></tr>`

function buildFile(name: string, holdings: string): File {
  const html = ACCOUNT_HEADER + holdings + HOLDINGS_FOOTER
  return new File([html], name, { type: 'text/html' })
}

export function createSampleFiles(): File[] {
  return [
    buildFile('샘플_ISA계좌.xls', ISA_HOLDINGS),
    buildFile('샘플_종합계좌.xls', JONGHAP_HOLDINGS),
    buildFile('샘플_연금저축계좌.xls', PENSION_HOLDINGS),
  ]
}

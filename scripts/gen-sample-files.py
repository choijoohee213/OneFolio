#!/usr/bin/env python3
"""테스트용 잔고파일 3개를 만든다. 실제 미래에셋 파일과 같은 구조·같은 함정을 담되
계좌번호와 금액은 전부 가짜다. 계좌 총합 3억.

    python3 scripts/gen-sample-files.py [출력폴더]

숫자는 손으로 적지 않고 수량·현재가·평단에서 계산한다. 매입금액·평가금액·손익·
손익률이 서로 어긋나면 테스트 파일로서 쓸모가 없다. 출금가능액도 자산총액에서
종목 평가합계를 빼서 자동으로 맞춘다.
"""
import sys

OUT = sys.argv[1] if len(sys.argv) > 1 else "res"

# 계좌 총합 3억. ISA·연금저축은 납입한도가 있어 종합계좌가 가장 크다.
TOTALS = {
    "ISA": ("111-1111-1111-0", "ISA(중개형)", 45_000_000),
    "종합": ("222-2222-2222-0", "종합", 195_000_000),
    "연금": ("333-3333-3333-0", "연금저축계좌(신)", 60_000_000),
}


def holding(name, qty, price, avg, foreign=False):
    """국내는 정수, 해외는 소수 2자리 — 원화 환산 때문에 소수점이 남는다."""
    d = 2 if foreign else 0
    buy = qty * avg
    evaluation = round(qty * price, 2)
    profit = evaluation - buy
    return {
        "eval": evaluation,
        "cells": [
            name,
            f"{qty:,.2f}",
            f"{price:,.{d}f}",
            f"{avg:,.{d}f}",
            f"{buy:,.{d}f}",
            f"{evaluation:,.{d}f}",
            f"{profit:,.{d}f}",
            f"{profit / buy * 100:.2f}%",
        ],
    }


# 외화 마이너스 잔고. 평단·손익·손익률이 "-" 이고 수량·금액이 음수인 실제 함정.
# 파서가 현금성 행으로 따로 빼므로 종목 합계에는 넣지 않는다.
FOREIGN_CASH = {
    "eval": 0,
    "cells": ["미국달러", "-30.500000", "1,450.10", "-", "-44,228.00", "-44,228.00", "-", "-"],
}

FILES = {
    "테스트_ISA계좌.xls": ("ISA", "111-11-1111110", [
        holding("삼성전자", 200, 75_000, 70_000),
        holding("TIGER 미국S&P500", 500, 21_500, 20_000),
        holding("TIGER 미국나스닥100", 100, 105_000, 100_000),
        holding("KODEX 레버리지", 300, 24_000, 25_000),
    ]),
    "테스트_종합계좌.xls": ("종합", "222-22-2222220", [
        holding("SK하이닉스", 150, 400_000, 380_000),
        holding("삼성전자", 400, 75_000, 72_000),
        holding("AMD", 120, 250_000.50, 240_000.00, foreign=True),
        holding("애플", 80, 320_000.25, 300_000.00, foreign=True),
        holding("알파벳 A", 52, 280_000.75, 260_000.00, foreign=True),
        holding("PROSHARES QQQ 3X", 200, 95_000.40, 100_000.00, foreign=True),
        holding("DIREXION SEMICONDUCTOR DAILY 3X", 60, 165_000.30, 180_000.00, foreign=True),
        FOREIGN_CASH,
    ]),
    "테스트_연금저축계좌.xls": ("연금", "333-33-3333330", [
        holding("TIGER 미국S&P500", 1_500, 21_500, 18_000),
        holding("SOL 미국배당다우존스", 1_200, 12_000, 11_000),
        holding("KODEX 미국AI전력핵심인프라", 300, 23_750, 25_000),
        holding("TIME 미국나스닥100액티브", 100, 45_000, 42_000),
    ]),
}

# 출금가능액은 손으로 적지 않는다. 자산총액에서 그 계좌 종목 평가합계를 뺀 값이
# 곧 앱이 현금성으로 잡는 금액이라, 둘이 어긋나면 테스트 데이터가 거짓말을 한다.
WITHDRAWABLE = {
    key: TOTALS[key][2] - sum(h["eval"] for h in holdings)
    for key, _, holdings in ((k, n, h) for n, (k, _, h) in FILES.items())
}

ACCOUNT_HEADERS = ["계좌번호", "계좌유형", "계좌별명", "자산총액", "출금가능액", "바로가기"]
HOLDING_HEADERS = ["상품명", "보유수량", "현재가", "평균매입가", "매입금액", "평가금액", "평가손익", "손익률"]
SHORTCUT = "계좌상세\t\t이체"


def row(cells, tag="td"):
    return "".join(f"<{tag}>{c.replace('&', '&amp;')}</{tag}>" for c in cells)


def build(account_no, holdings):
    accounts = "\n".join(
        "\t\t\t<tr>"
        + row([no, kind, "", f"{total:,}", f"{WITHDRAWABLE[key]:,.0f}", SHORTCUT])
        + "</tr>"
        for key, (no, kind, total) in TOTALS.items()
    )
    rows = "\n".join("\t\t\t<tr>" + row(h["cells"]) + "</tr>" for h in holdings)

    return f"""<html>
\t<head>
\t\t<meta http-equiv="Content-Type" content="text/html; charset=utf-8">
\t</head>
\t<body>
\t\t<table><tr><td height="30" colspan="8">계좌별잔고</td></tr></table>
\t\t<table><tr><td></td></tr></table>
\t\t<table><tr><td>전체 계좌현황</td><td colspan="4">(단위: 원)</td></tr></table>
\t\t<table class="col_type" summary="계좌번호, 계좌유형, 계좌별명, 자산총액, 출금가능액, 바로가기로 구성된 표 입니다.">
\t\t\t<tr>{row(ACCOUNT_HEADERS, 'th')}</tr>
{accounts}
\t\t</table>
\t\t<table><tr><td></td></tr></table>
\t\t<table><tr><td colspan="7">[{account_no}] 상품보유현황</td></tr></table>
\t\t<table class="col_type" summary="상품명, 보유수량, 현재가, 평균매입가, 매입금액, 평가금액, 평가손익, 손익률로 구성된 표 입니다.">
\t\t\t<tr>{row(HOLDING_HEADERS, 'th')}</tr>
{rows}
\t\t</table>
\t</body>
</html>
"""


for filename, (key, account_no, holdings) in FILES.items():
    with open(f"{OUT}/{filename}", "w", encoding="utf-8") as f:
        f.write(build(account_no, holdings))
    total = sum(h["eval"] for h in holdings)
    print(f"{filename:26s} 종목 {len(holdings)}개  평가합계 {total:>15,.0f}  예수금 {WITHDRAWABLE[key]:>12,.0f}")

print()
print(f"{'계좌 총합':<26s} {sum(t for _, _, t in TOTALS.values()):>21,}")
for key, (_, kind, total) in TOTALS.items():
    print(f"  {kind:<20s} {total:>14,}")

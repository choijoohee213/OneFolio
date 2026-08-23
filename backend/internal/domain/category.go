package domain

type Category string

const (
	DomesticStock Category = "개별주(국내)"
	ForeignStock  Category = "개별주(해외)"
	IndexETF      Category = "지수 ETF"
	ThemeETF      Category = "테마·섹터 ETF"
	Bond          Category = "채권"
	Leverage      Category = "레버리지·인버스"
	Cash          Category = "현금성"
)

func Categories() []Category {
	return []Category{DomesticStock, ForeignStock, IndexETF, ThemeETF, Bond, Leverage, Cash}
}

func (c Category) Valid() bool {
	for _, known := range Categories() {
		if c == known {
			return true
		}
	}
	return false
}

// legacyCategories 는 예전 이름으로 저장된 사용자 지정 분류를 옮긴다. 레버리지와
// 테마가 한 칸이던 시절의 값이라 어느 쪽인지 알 수 없는데, 레버리지는 이름으로
// 잘 가려지니 테마 쪽으로 보낸다 — 틀렸으면 자동 분류가 다시 잡아 준다.
var legacyCategories = map[Category]Category{
	"레버리지·테마 ETF": ThemeETF,
}

// Migrate 는 예전 이름이면 지금 쓰는 이름으로 바꾼다. 아니면 그대로 둔다.
func (c Category) Migrate() Category {
	if next, ok := legacyCategories[c]; ok {
		return next
	}
	return c
}

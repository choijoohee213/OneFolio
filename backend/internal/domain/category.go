package domain

type Category string

const (
	DomesticStock Category = "개별주(국내)"
	ForeignStock  Category = "개별주(해외)"
	IndexETF      Category = "지수 ETF"
	ThemeETF      Category = "레버리지·테마 ETF"
	Cash          Category = "현금성"
)

func Categories() []Category {
	return []Category{DomesticStock, ForeignStock, IndexETF, ThemeETF, Cash}
}

func (c Category) Valid() bool {
	for _, known := range Categories() {
		if c == known {
			return true
		}
	}
	return false
}

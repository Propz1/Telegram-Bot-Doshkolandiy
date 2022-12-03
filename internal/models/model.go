package models

type Warehouse struct {
	NameWarehouse string `json:"НаименованиеСклада"`
}

type Warehouses struct {
	Result         bool        `json:"Результат"`
	Information    string      `json:"Информация"`
	ListWarehouses []Warehouse `json:"ТаблицаДанных"`
}

type Remainder struct {
	Nomenclature string `json:"НаименованиеНоменклатуры"`
	Code         string `json:"КодНоменклатуры"`
	Store        string `json:"Склад"`
}

type RemainderQuantity struct {
	Nomenclature string  `json:"НаименованиеНоменклатуры"`
	Code         string  `json:"КодНоменклатуры"`
	Quantity     float64 `json:"Количество"`
}

type RemainderList []RemainderQuantity

type WarehouseRemainder struct {
	Result        bool                `json:"Результат"`
	Information   string              `json:"Информация"`
	RemainderList []RemainderQuantity `json:"ТаблицаДанных"`
}

type ArrayRemainder []Remainder

func (a ArrayRemainder) Len() int {
	return len(a)
}

func (a ArrayRemainder) Less(i, j int) bool {
	return a[i].Store < a[j].Store
}

func (a ArrayRemainder) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a RemainderList) Len() int {
	return len(a)
}

func (a RemainderList) Less(i, j int) bool {
	return a[i].Nomenclature < a[j].Nomenclature
}

func (a RemainderList) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

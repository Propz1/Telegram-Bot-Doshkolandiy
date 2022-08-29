package models

type Information struct {
	Result bool `json:"Результат"`
	Information string `json:"Информация"`
}

type Remainder struct {
	Nomenclature string `json:"НаименованиеНоменклатуры"`
	Code         string `json:"КодНоменклатуры"`
	Store        string `json:"Склад"`
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

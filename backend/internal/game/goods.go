package game

type GoodID string

type Good struct {
	ID   GoodID `json:"id"`
	Name string `json:"name"`
}

const (
	// StartingCredits is what a new player starts with.
	StartingCredits = 1000
	// DefaultCargoCapacity is how many units a new ship can carry; every good takes 1 unit.
	DefaultCargoCapacity = 20
)

// defaultGoods is the fixed catalog of goods, in display order.
func defaultGoods() []Good {
	return []Good{
		{ID: "food", Name: "Food"},
		{ID: "water", Name: "Water"},
		{ID: "ore", Name: "Ore"},
		{ID: "metals", Name: "Metals"},
		{ID: "fuel", Name: "Fuel"},
		{ID: "textiles", Name: "Textiles"},
		{ID: "medicine", Name: "Medicine"},
		{ID: "electronics", Name: "Electronics"},
		{ID: "machinery", Name: "Machinery"},
		{ID: "luxuries", Name: "Luxuries"},
	}
}

// Good returns the good with the given ID.
func (w *World) Good(id GoodID) (Good, bool) {
	i, ok := w.goodIdx[id]
	if !ok {
		return Good{}, false
	}
	return w.Goods[i], true
}

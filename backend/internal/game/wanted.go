package game

// Wanted heat: every successful plunder makes the player wanted for a while.
// Heat is the remaining wanted time in ticks; everyone can see it. Above
// RefuseTradeHeat, stations refuse to trade with the player.
const (
	HeatPerPlunder  = 120 // ticks of wanted time added by each plunder
	MaxHeat         = 600
	RefuseTradeHeat = 180 // above this, stations won't trade with you
)

// Wanted reports whether the player is marked as a pirate.
func (p *Player) Wanted() bool { return p.Heat > 0 }

// addHeat makes the player wanted for HeatPerPlunder more ticks, up to MaxHeat.
func (p *Player) addHeat() { p.Heat = min(p.Heat+HeatPerPlunder, MaxHeat) }

// coolDown lowers every player's heat by one tick; called from Advance.
func (w *World) coolDown() {
	for _, p := range w.Players {
		if p.Heat > 0 {
			p.Heat--
		}
	}
}

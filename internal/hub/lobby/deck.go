package lobby

import (
	"math/rand"

	"github.com/Nerdmaster/poker"
)

type Deck struct {
	rnd    *rand.Rand
	offset int
	cards  poker.CardList
}

func NewDeck(rndSource rand.Source) *Deck {
	var d = &Deck{rnd: rand.New(rndSource)}
	d.cards = make(poker.CardList, 52)
	d.offset = 1
	var idx = 0
	for rank := poker.Deuce; rank <= poker.Ace; rank++ {
		for _, suit := range []poker.CardSuit{poker.Spades, poker.Hearts, poker.Diamonds, poker.Clubs} {
			d.cards[idx] = poker.NewCard(rank, suit)
			idx++
		}
	}
	return d
}

func (d *Deck) Shuffle() {
	d.rnd.Shuffle(len(d.cards), func(i, j int) {
		d.cards[i], d.cards[j] = d.cards[j], d.cards[i]
	})
	d.offset = 1
}

func (d *Deck) Draw(it int) func(yield func(poker.Card) bool) {
	return func(yield func(poker.Card) bool) {
		for range it {
			if d.offset > len(d.cards) {
				return
			}
			yield(d.cards[len(d.cards)-d.offset])
			d.offset++
		}
	}

}

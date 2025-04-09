package lobby

import (
	"math/rand/v2"

	"github.com/Nerdmaster/poker"
)

type Deck struct {
	rnd    *rand.Rand
	offset int
	cards  poker.CardList
}

func NewDeck(src rand.Source) *Deck {
	var d = &Deck{rnd: rand.New(src)}
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

func (d *Deck) Draw(cards poker.CardList) {
	for i := range cards {
		if d.offset > len(d.cards) {
			return
		}
		cards[i] = d.cards[len(d.cards)-d.offset]
		d.offset++
	}
}

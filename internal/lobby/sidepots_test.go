package lobby_test

import (
	"testing"

	"github.com/retinotopic/go-bet/internal/lobby"
	"github.com/stretchr/testify/assert"
)

func TestTwoPlayersDifferentBets(t *testing.T) {
	users := []lobby.PlayUnit{
		{Bet: 100, Bank: 0}, // Игрок 0
		{Bet: 50, Bank: 0},  // Игрок 1
	}
	topPlaces := []lobby.Top{
		{Place: 0, Exiled: false, Rating: 2},
		{Place: 1, Exiled: false, Rating: 1},
	}
	lobby := &lobby.Lobby{
		AllUsers:  users,
		TopPlaces: topPlaces,
	}

	lobby.CalcSidePots(100, 50)

	// Проверка игрока 0
	assert.Equal(t, 150, lobby.AllUsers[0].Bank, "Игрок 0 должен получить 50 в банк")
	// assert.Equal(t, 0, lobby.AllUsers[0].SidePots[0], "Сайд-пот игрока 0 должен быть обнулен")
	assert.False(t, lobby.TopPlaces[0].Exiled, "Игрок 0 не должен быть исключен")

	// Проверка игрока 1
	assert.Equal(t, 0, lobby.AllUsers[1].Bank, "Игрок 1 не должен получить деньги")
	assert.True(t, lobby.TopPlaces[1].Exiled, "Игрок 1 должен быть исключен")
}
func TestThreePlayersMultiplePots(t *testing.T) {
	users := []lobby.PlayUnit{
		{Bet: 150, Bank: 0},
		{Bet: 100, Bank: 0},
		{Bet: 50, Bank: 0},
	}
	topPlaces := []lobby.Top{
		{Place: 0, Rating: 3},
		{Place: 1, Rating: 2},
		{Place: 2, Rating: 1},
	}
	lobby := &lobby.Lobby{AllUsers: users, TopPlaces: topPlaces}

	lobby.CalcSidePots(150, 100)

	assert.Equal(t, 300, lobby.AllUsers[0].Bank, "Игрок 0 должен получить 300")
	assert.Equal(t, 0, lobby.AllUsers[1].Bank, "Игрок 1 должен получить 0")
	assert.Equal(t, 0, lobby.AllUsers[2].Bank, "Игрок 2 должен получить 0")
}
func TestMultPlayersMultiplePots(t *testing.T) {
	users := []lobby.PlayUnit{
		{Bet: 150, Bank: 0},
		{Bet: 900, Bank: 0},
		{Bet: 700, Bank: 0},
		{Bet: 360, Bank: 0},
	}
	topPlaces := []lobby.Top{
		{Place: 3, Rating: 3},
		{Place: 1, Rating: 2},
		{Place: 0, Rating: 1},
		{Place: 2, Rating: 0},
	}
	lobby := &lobby.Lobby{AllUsers: users, TopPlaces: topPlaces}

	lobby.CalcSidePots(900, 700)

	assert.Equal(t, 1230, lobby.AllUsers[3].Bank, "Игрок 3 должен получить 1230")
	assert.Equal(t, 880, lobby.AllUsers[1].Bank, "Игрок 1 должен получить 880")
	assert.Equal(t, 0, lobby.AllUsers[0].Bank, "Игрок 0 должен получить 0")
	assert.Equal(t, 0, lobby.AllUsers[2].Bank, "Игрок 2 должен получить 0")
}

func TestMultiplePotsAndSplit(t *testing.T) {
	users := []lobby.PlayUnit{
		{Bet: 440, Bank: 0},
		{Bet: 720, Bank: 0},
		{Bet: 530, Bank: 0},
	}
	topPlaces := []lobby.Top{
		{Place: 0, Rating: 3},
		{Place: 1, Rating: 2},
		{Place: 2, Rating: 2},
	}
	lobby := &lobby.Lobby{AllUsers: users, TopPlaces: topPlaces}

	lobby.CalcSidePots(720, 530)

	assert.Equal(t, 1320, lobby.AllUsers[0].Bank, "Игрок 0 должен получить 1320")
	assert.Equal(t, 280, lobby.AllUsers[1].Bank, "Игрок 1 должен получить 280")
	assert.Equal(t, 90, lobby.AllUsers[2].Bank, "Игрок 2 должен получить 90")
}
func TestEqualRatingSplit(t *testing.T) {
	users := []lobby.PlayUnit{
		{Bet: 100, Bank: 0},
		{Bet: 100, Bank: 0},
	}
	topPlaces := []lobby.Top{
		{Place: 0, Rating: 2},
		{Place: 1, Rating: 2},
	}
	lobby := &lobby.Lobby{AllUsers: users, TopPlaces: topPlaces}

	lobby.CalcSidePots(100, 0)

	assert.Equal(t, 100, lobby.AllUsers[0].Bank, "Игрок 0 должен получить 50")
	assert.Equal(t, 100, lobby.AllUsers[1].Bank, "Игрок 1 должен получить 50")
}

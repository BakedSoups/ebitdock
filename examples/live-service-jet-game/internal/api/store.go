package api

import (
	"sync"

	"example.com/orbit-snake/internal/shared"
)

type Store struct {
	mu      sync.RWMutex
	players map[string]shared.PlayerProfile
}

func NewStore() *Store {
	return &Store{players: map[string]shared.PlayerProfile{}}
}

func (s *Store) Player(id string) shared.PlayerProfile {
	s.mu.Lock()
	defer s.mu.Unlock()
	if player, ok := s.players[id]; ok {
		return player
	}
	player := shared.PlayerProfile{
		ID:    id,
		Name:  "pilot-" + id,
		Color: "#4bd9ce",
		Upgrades: shared.Upgrades{
			SpeedLevel: 1,
			TurnLevel:  1,
			BoostLevel: 1,
		},
	}
	s.players[id] = player
	return player
}

func (s *Store) AddScrap(id string, amount int) shared.PlayerProfile {
	s.mu.Lock()
	defer s.mu.Unlock()
	player, ok := s.players[id]
	if !ok {
		player = shared.PlayerProfile{ID: id, Name: "pilot-" + id, Color: "#4bd9ce"}
	}
	player.TotalScrap += amount
	if player.TotalScrap < 0 {
		player.TotalScrap = 0
	}
	s.players[id] = player
	return player
}

func (s *Store) Leaderboard() []shared.PlayerProfile {
	s.mu.RLock()
	defer s.mu.RUnlock()
	players := make([]shared.PlayerProfile, 0, len(s.players))
	for _, player := range s.players {
		players = append(players, player)
	}
	return players
}

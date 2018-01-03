package main

import (
	elo "playerElo"
	"github.com/satori/go.uuid"
)

func main() {
	rankings, err := elo.NewPlayerRankings(
		"postgres://postgres:mysecretpassword@localhost/player_elo?sslmode=disable",
		1200, 24, 10,
	)
	if err != nil {
		panic(err)
	}
	hack_uuid, _ := uuid.FromString("597d1b49-f7a9-483e-8e93-b0b4bfa50991")
	bigmac_uuid, _ := uuid.FromString("9c5d87b6-e342-4fb8-953b-88487aaf4c6d")
	err = rankings.RecordMatch(hack_uuid, bigmac_uuid, false)
	if err != nil {
		panic(err)
	}
}

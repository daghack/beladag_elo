package playerElo

import (
//	"database/sql"
	"fmt"
	"time"
	"math"
	"github.com/jmoiron/sqlx"
	"github.com/satori/go.uuid"
	_ "github.com/lib/pq"
)

const (
	KIT_Arc = "Archery"
	KIT_Blu = "SingleBlue"
	KIT_Flo = "Florentine"
	KIT_Red = "SingleRed"
	KIT_SnB = "SwordAndBoard"
	KIT_SnS = "SwordAndStaff"
	KIT_Spr = "SingleSpear"
)

const (
	createPgCryptoStr = `CREATE EXTENSION IF NOT EXISTS pgcrypto`
	createTableStr = `CREATE TABLE IF NOT EXISTS player_elo (
		uuid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		game_name TEXT NOT NULL,
		kit TEXT NOT NULL,
		matches BIGINT DEFAULT 0 NOT NULL,
		rating DOUBLE PRECISION NOT NULL
	)`
	getAvgStr = `SELECT avg(rating) FROM (SELECT rating FROM player_elo) AS average`
	newPlayerStr = `INSERT INTO player_elo (game_name, kit, rating) VALUES ($1, $2, $3)`
	fetchPlayerByUUID = `SELECT * FROM player_elo WHERE uuid = $1`
	fetchPlayersByKit = `SELECT * FROM player_elo WHERE kit = $1 ORDER BY rating DESC`
	fetchPlayersByGameName = `SELECT * FROM player_elo WHERE game_name = $1`
	addNewMatchStr = `UPDATE player_elo SET matches = matches+1, rating = $2 WHERE uuid = $1`
)

type Player struct {
	UUID uuid.UUID `db:"uuid"`
	GameName string `db:"game_name"`
	Kit string `db:"kit"`
	Matches int `db:"matches"`
	Rating float64 `db:"rating"`
}

type averageResp struct {
	Average float64 `db:"average"`
}

type PlayerRanking struct {
	db *sqlx.DB
	kFactor float64
	defaultRanking float64
	provisionalMatches int
}

func NewPlayerRankings(dbstring string, defaultRanking, kFactor float64, provMatches int) (*PlayerRanking, error) {
	db, err := sqlx.Connect("postgres", dbstring)
	connected := false
	for ttw := 1; ttw < 10; ttw *= 2 {
		db, err = sqlx.Connect("postgres", dbstring)
		if err != nil {
			fmt.Printf("Could not connect, retrying in %d seconds.\n", ttw)
			time.Sleep(time.Duration(ttw) * time.Second)
		} else {
			connected = true
			break
		}
	}
	if !connected {
		return nil, err
	}
	pinged := false
	for ttw := 1; ttw < 10; ttw *= 2 {
		err = db.Ping()
		if err != nil {
			fmt.Printf("Could not ping, retrying in %d seconds.\n", ttw)
			time.Sleep(time.Duration(ttw) * time.Second)
		} else {
			pinged = true
			break
		}
	}
	if !pinged {
		return nil, err
	}
	_, err = db.Exec(createPgCryptoStr)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(createTableStr)
	if err != nil {
		return nil, err
	}
	return &PlayerRanking{
		db : db,
		kFactor : kFactor,
		defaultRanking : defaultRanking,
		provisionalMatches : provMatches,
	}, nil
}

func (elosys *PlayerRanking) NewPlayer(gamename, kit string) error {
	startscore := elosys.defaultRanking
	if startscore == 0 {
		avg := averageResp{}
		err := elosys.db.Get(&avg, getAvgStr)
		if err != nil {
			return err
		}
		startscore = avg.Average
	}
	_, err := elosys.db.Exec(newPlayerStr, gamename, kit, startscore)
	return err
}

func (elosys *PlayerRanking) RecordMatch(winner, loser uuid.UUID, draw bool) error {
	wplayer := &Player{}
	lplayer := &Player{}
	err := elosys.db.Get(wplayer, fetchPlayerByUUID, winner)
	if err != nil {
		return err
	}
	err = elosys.db.Get(lplayer, fetchPlayerByUUID, loser)
	if err != nil {
		return err
	}
	fmt.Printf("%v\n%v\n", wplayer, lplayer)
	wdelta, ldelta := FindDeltas(
		wplayer.Rating, lplayer.Rating, draw,
		wplayer.Matches <= elosys.provisionalMatches, lplayer.Matches <= elosys.provisionalMatches,
		elosys.kFactor,
	)
	fmt.Printf("New Ratings: %f, %f\n", wplayer.Rating + wdelta, lplayer.Rating + ldelta)
	_, err = elosys.db.Exec(addNewMatchStr, wplayer.UUID, wplayer.Rating + wdelta)
	if err != nil {
		return err
	}
	_, err = elosys.db.Exec(addNewMatchStr, lplayer.UUID, lplayer.Rating + ldelta)
	return err
}

func (elosys *PlayerRanking) FetchByKit(kit string) ([]Player, error) {
	toret := []Player{}
	err := elosys.db.Select(&toret, fetchPlayersByKit, kit)
	if err != nil {
		return toret, err
	}
	return toret, nil
}

func ExpectedScore(p1, p2 float64) (float64, float64) {
	R1 := float64(math.Pow10(int(p1/400.0)))
	R2 := float64(math.Pow10(int(p2/400.0)))
	fmt.Printf("Expected Score: %f, %f\n", R1/(R1 + R2), R2/(R1 + R2))
	return (R1 / (R1 + R2)), (R2 / (R1 + R2))
}

func FindDeltas(winner, loser float64, draw bool, winner_p, loser_p bool, kFactor float64) (float64, float64) {
	score1 := 1.0
	score2 := 0.0
	if draw {
		score1 = 0.5
		score2 = 0.5
	}
	provisional1 := 1.0
	provisional2 := 1.0
	if winner_p != loser_p {
		if winner_p {
			provisional1 = 0.0
		} else {
			provisional2 = 0.0
		}
	}
	fmt.Printf("Score: %f, %f\n", score1, score2)
	expected1, expected2 := ExpectedScore(winner, loser)
	//If the loser is provisional and the winner is not, then the winner gains no points.
	wdelta := kFactor * provisional2 * (score1 - expected1)
	//If the winner is provisional and the loser is not, then the loser loses no points.
	ldelta := kFactor * provisional1 * (score2 - expected2)
	fmt.Printf("KFactor: %f\nProvisionals: %f, %f\nDeltas: %f, %f\n", kFactor, provisional1, provisional2, wdelta, ldelta)
	return wdelta, ldelta
}

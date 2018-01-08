package main

import (
	"fmt"
	"github.com/kelseyhightower/envconfig"
	"github.com/satori/go.uuid"
	elo "playerElo"
)

type PostgresConf struct {
	Username string
	Password string
	Host string
	Database string
}

type Conf struct {
	PostgresConf
}

func main() {
	var conf Conf
	err := envconfig.Process("dag_ratings", &conf)
	if err != nil {
		panic(err)
	}
	db_url := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", conf.Username, conf.Password, conf.Host, conf.Database)
	rankings, err := elo.NewPlayerRankings(db_url, 1200, 24, 10)
	if err != nil {
		panic(err)
	}
	hack_uuid, _ := uuid.FromString("597d1b49-f7a9-483e-8e93-b0b4bfa50991")
	bigmac_uuid, _ := uuid.FromString("9c5d87b6-e342-4fb8-953b-88487aaf4c6d")
	err = rankings.RecordMatch(hack_uuid, bigmac_uuid, false)
	if err != nil {
		panic(err)
	}
	fmt.Println("SUCCESS")
}

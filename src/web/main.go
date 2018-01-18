package main

import (
	"github.com/kelseyhightower/envconfig"
	"pages"
)

func main() {
	var conf pages.Conf
	envconfig.Process("bdr", &conf)
	wh, err := pages.NewWebHandler(&conf)
	if err != nil {
		panic(err)
	}
	wh.Serve()
}

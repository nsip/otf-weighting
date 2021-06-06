package config

import (
	"fmt"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/cdutwhu/debog/fn"
	"github.com/davecgh/go-spew/spew"
)

func TestConfig(t *testing.T) {
	cfg := &Config{}
	_, err := toml.DecodeFile("./config.toml", cfg)
	fn.FailOnErr("%v", err)
	fmt.Println("-------------------------------")
	spew.Dump(cfg)
}

func TestGetConfig(t *testing.T) {
	cfg := GetConfig("./config.toml")
	spew.Dump(cfg)
}

func TestOthers(t *testing.T) {
	m := make(map[string]string)
	if v, ok := m["a"]; v == "" && !ok {
		fmt.Println("default empty string if !ok")
	}
}

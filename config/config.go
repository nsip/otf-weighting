package config

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

// Config :
type Config struct {
	FatalOnErr  bool
	InType      string
	MustInArray bool
	In          string
	InTemp      string
	OutType     string
	Out         string
	Service     struct {
		Port int
		API  string
	}
	Weighting struct {
		StudentIDPath        string
		ProgressionLevelPath string
		TimePath0            string
		TimePath1            string
		ScorePath            string
	}
}

// GetConfig :
func GetConfig(configs ...string) *Config {
	for _, config := range configs {
		cfg := &Config{}
		_, err := toml.DecodeFile(config, cfg)
		if err != nil {
			continue
		}

		// Directory Process
		cfg.In = filepath.Clean(cfg.In)
		cfg.Out = filepath.Clean(cfg.Out)

		// API Process
		cfg.Service.API = withSlash(cfg.Service.API)

		// Env
		os.Setenv("FatalOnErr", strconv.FormatBool(cfg.FatalOnErr))

		return cfg
	}
	log.Fatalln("OTF Weighting Config File is Missing or Error")
	return nil
}

func withSlash(str string) string {
	return "/" + strings.Trim(str, "/")
}

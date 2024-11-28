package main

import (
	"flag"
	"io"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/k0kubun/pp/v3"
)

func main() {
	var configPath string

	flag.StringVar(&configPath, "c", "cel.toml", "path to config file")

	flag.Parse()

	file, err := os.Open(configPath)
	if err != nil {
		panic("Config file not found!")
	}

	configData, err := io.ReadAll(file)

	if err != nil {
		panic("Failed to read config file!")
	}

	config := Config{}
	err = toml.Unmarshal(configData, &config)
	if err != nil {
		panic(err)
	}

	pp.Print(config)
}

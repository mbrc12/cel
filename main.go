package main

import (
	"flag"
	"io"
	"os"
)

type Model struct {
	Config *Config

	Current      string
	TaskIndex    int
	SubTaskIndex int
}

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

	config := new(Config)
	config.Parse(configData)

	model := Model{
		Config:       config,
		Current:      "idle",
		TaskIndex:    -1,
		SubTaskIndex: 0,
	}

	_ = model
}

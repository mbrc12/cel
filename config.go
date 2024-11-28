package main

import (
	"errors"

	"github.com/BurntSushi/toml"
	"github.com/k0kubun/pp/v3"
)

type Config struct {
	Prefix     string      `toml:"prefix"`
	WatchTasks []WatchTask `toml:"watch"`
	MenuTasks  []MenuTask  `toml:"menu"`
}

func (self *Config) Parse(data []byte) error {
	err := toml.Unmarshal(data, &self)
	if err != nil {
		return err
	}

	id := 0
	for i := range self.WatchTasks {
		self.WatchTasks[i].Id = id
		id++
	}

	for i := range self.MenuTasks {
		self.MenuTasks[i].Id = id
		id++
	}

	pp.Println(self)

	return nil
}

type WatchTask struct {
	Id    int
	Files []string  `toml:"files"`
	Run   []Command `toml:"run"`
}

type MenuTask struct {
	Id  int
	Key string    `toml:"key"`
	Run []Command `toml:"run"`
}

type Command struct {
	Commands []string
}

func (self *Command) UnmarshalTOML(data any) error {
	switch data.(type) {
	case string:
		self.Commands = []string{data.(string)}
	case []any:
		rawArr := data.([]any)
		if len(rawArr) == 0 {
			return errors.New("Command array cannot be empty")
		}
		self.Commands = make([]string, len(rawArr))
		for i, v := range rawArr {
			cmd, ok := v.(string)
			if !ok {
				return errors.New("Command array can only contain strings")
			}
			self.Commands[i] = cmd
		}
	default:
		return errors.New("Command has to be a string or an array of strings")
	}
	return nil
}

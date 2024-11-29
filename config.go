package main

import (
	"errors"
	"strconv"

	"github.com/BurntSushi/toml"
)

var (
	defaultPrefix    = []string{"bash", "-c"}
	defaultStoreSize = StoreSize(1024 * 1024) // 1MB
)

type Config struct {
	Prefix     []string    `toml:"prefix"`
	Store      StoreSize   `toml:"store"`
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
		if len(self.WatchTasks[i].Run.Commands) == 0 {
			return errors.New("Watch task command cannot be empty!")
		}
		id++
	}

	for i := range self.MenuTasks {
		self.MenuTasks[i].Id = id
		if len(self.MenuTasks[i].Run.Commands) == 0 {
			return errors.New("Menu task command cannot be empty!")
		}
		id++
	}

	if self.Prefix == nil {
		self.Prefix = defaultPrefix
	}

	if self.Store == StoreSize(0) {
		self.Store = defaultStoreSize
	}

	return nil
}

type WatchTask struct {
	Id      int
	Files   []string `toml:"files"`
	Exclude []string `toml:"exclude"`
	Run     Command  `toml:"run"`
}

type MenuTask struct {
	Id  int
	Key string  `toml:"key"`
	Run Command `toml:"run"`
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

type StoreSize uint64

func (self *StoreSize) UnmarshalText(data []byte) error {
	s := string(data)
	base, err := strconv.ParseInt(s[:len(s)-1], 10, 64)
	if err != nil {
		return err
	}

	switch s[len(s)-1] {
	case 'K':
		base *= 1024
	case 'M':
		base *= 1024 * 1024
	case 'G':
		base *= 1024 * 1024 * 1024
	case 'B':
		// do nothing
	default:
		last_digit := int64(s[len(s)-1] - '0')
		if last_digit >= 0 && last_digit <= 9 {
			base *= 10
			base += last_digit
		} else {
			return errors.New("Invalid size suffix")
		}
	}

	*self = StoreSize(base)

	return nil
}

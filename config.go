package main

import (
	"errors"
)

type Config struct {
	Prefix     string      `toml:"prefix"`
	WatchTasks []WatchTask `toml:"watch"`
	MenuTasks  []MenuTask  `toml:"menu"`
}

type WatchTask struct {
	Watch []string  `toml:"watch"`
	Run   []Command `toml:"run"`
}

type MenuTask struct {
	Button string    `toml:"button"`
	Run    []Command `toml:"run"`
}

type Command struct {
	Commands []string
}

func (c *Command) UnmarshalTOML(data any) error {
	switch data.(type) {
	case string:
		c.Commands = []string{data.(string)}
	case []any:
		rawArr := data.([]any)
		if len(rawArr) == 0 {
			return errors.New("Command array cannot be empty")
		}
		c.Commands = make([]string, len(rawArr))
		for i, v := range rawArr {
			cmd, ok := v.(string)
			if !ok {
				return errors.New("Command array can only contain strings")
			}
			c.Commands[i] = cmd
		}
	default:
		return errors.New("Command has to be a string or an array of strings")
	}
	return nil

	// value := reflect.ValueOf(data)
	// println(value.Kind())
	// switch value.Kind() {
	//
	// case reflect.String:
	// 	c.Commands = []string{data.(string)}
	//
	// case reflect.Slice:
	// 	println(value.Len())
	// 	if value.Len() == 0 {
	// 		return errors.New("Command array cannot be empty")
	// 	}
	// 	c.Commands = make([]string, value.Len())
	// 	for i := 0; i < value.Len(); i++ {
	// 		cmd, ok := value.Index(i).Interface().(string)
	// 		if !ok {
	// 			return errors.New("Command array can only contain strings")
	// 		}
	// 		c.Commands[i] = cmd
	// 	}
	//
	// default:
	// 	return errors.New("Command has to be a string or an array of strings")
	// }
	//
	// return nil
}

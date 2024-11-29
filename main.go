package main

import (
	"flag"
	"io"
	"os"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	Config *Config

	TaskIndex  int
	Restarting bool

	OutputSinks map[int]chan string
	Outputs     map[int]string

	ControlChans map[int]chan TaskCmd

	Tasks map[int]*Task

	UIStates map[int]UIState
}

type UIState struct {
	Viewport viewport.Model
	Data     string
}

func main() {
	var configPath string

	flag.StringVar(&configPath, "c", "stag.toml", "path to config file")

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

	sinks := make(map[int]chan string)
	control := make(map[int]chan TaskCmd)
	tasks := make(map[int]*Task)

	setupTask := func(id int, commands []string, menuTask bool, setupFn func(task *Task)) {
		task := &Task{
			OutputChan: sinks[id],
			Prefix:     config.Prefix,
			Commands:   commands,
			IsMenuTask: menuTask,
		}

		task.Init()

		tasks[id] = task

		var taskEvts chan TaskCmd
		control[id] = taskEvts

		setupFn(task)

		go func() { task.Start(taskEvts) }()
	}

	for _, taskConfig := range config.WatchTasks {
		setupTask(taskConfig.Id, taskConfig.Run.Commands, false, func(task *Task) {
			task.Watch(taskConfig.Files, taskConfig.Exclude)
		})
	}

	for _, taskConfig := range config.MenuTasks {
		setupTask(taskConfig.Id, taskConfig.Run.Commands, true, func(task *Task) {})
	}

	model := Model{
		Config:    config,
		TaskIndex: 0,

		OutputSinks:  sinks,
		ControlChans: control,

		Restarting: false,

		Tasks:    tasks,
		UIStates: make(map[int]UIState),
	}

	program := tea.NewProgram(&model)
	if _, err := program.Run(); err != nil {
		panic(err)
	}
}

type newOutputLine struct {
	id   int
	line string
}

// sinkWatcher watches a sink channel and sends a message to the update loop
// when it receives a new line.
func sinkWatcher(id int, sink chan string) tea.Cmd {
	return func() tea.Msg {
		println("sinkWatcher")
		return newOutputLine{id, <-sink}
	}
}

func (self *Model) Init() tea.Cmd {
	cmds := make([]tea.Cmd, 0)

	for id, c := range self.OutputSinks {
		cmds = append(cmds, sinkWatcher(id, c))
	}

	return tea.Batch(cmds...)
}

func (self *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return self, tea.Quit
		}

	case newOutputLine:
		self.Outputs[msg.id] += msg.line + "\n"

		// trim to size
		maxSize := int(self.Config.Store)
		if len(self.Outputs[msg.id]) > maxSize {
			self.Outputs[msg.id] = self.Outputs[msg.id][len(self.Outputs[msg.id])-maxSize:]
		}
	}

	return self, nil
}

func (self *Model) View() string {
	return string(self.Outputs[self.TaskIndex])
}

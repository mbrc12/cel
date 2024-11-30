package main

import (
	"flag"
	"io"
	"os"
	"regexp"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/k0kubun/pp/v3"
)

var (
	log = pp.Println
)

type Model struct {
	Config *Config

	TaskIndex  int
	Restarting bool

	OutputSinks map[int]chan string
	Outputs     map[int]string

	ControlChans map[int]chan TaskCmd

	Tasks    map[int]*Task
	UIStates map[int]*UIState
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
	err = config.Parse(configData)
	if err != nil {
		panic(err)
	}

	sinks := make(map[int]chan string)
	control := make(map[int]chan TaskCmd)
	tasks := make(map[int]*Task)

	setupTask := func(id int, commands []string, menuTask bool, logFile string, setupFn func(task *Task)) {
		sinks[id] = make(chan string) // dont buffer output

		task := &Task{
			Format:   config.Format,
			Commands: commands,
			LogFile:  logFile,

			OutputChan: sinks[id],
			IsMenuTask: menuTask,
		}

		task.Init()

		tasks[id] = task

		taskEvts := make(chan TaskCmd, 10) // buffer 10 events from client
		control[id] = taskEvts

		setupFn(task)

		go func() { task.Start(taskEvts) }()
	}

	for _, taskConfig := range config.WatchTasks {
		setupTask(taskConfig.Id, taskConfig.Run.Commands, false, taskConfig.Log, func(task *Task) {
			task.Watch(taskConfig.Files, taskConfig.Exclude)
		})
	}

	for _, taskConfig := range config.MenuTasks {
		setupTask(taskConfig.Id, taskConfig.Run.Commands, true, taskConfig.Log, func(task *Task) {})
	}

	model := Model{
		Config:    config,
		TaskIndex: 0,

		OutputSinks:  sinks,
		ControlChans: control,

		Outputs: make(map[int]string),

		Restarting: false,

		Tasks:    tasks,
		UIStates: make(map[int]*UIState),
	}

	defer func() {
		print(model.Outputs[0])
	}()

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
func (self *Model) sinkWatcher(id int) tea.Cmd {
	return func() tea.Msg {
		line := <-self.OutputSinks[id]
		line = ansiSanitize(line)
		return newOutputLine{id, line}
	}
}

func (self *Model) Init() tea.Cmd {
	cmds := []tea.Cmd{}

	for id := range self.OutputSinks {
		cmds = append(cmds, self.sinkWatcher(id))
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
		self.Outputs[msg.id] += msg.line

		// trim to size
		maxSize := int(self.Config.Store)
		if len(self.Outputs[msg.id]) > maxSize {
			self.Outputs[msg.id] = self.Outputs[msg.id][len(self.Outputs[msg.id])-maxSize:]
		}

		return self, self.sinkWatcher(msg.id) // watch for more output
	}

	return self, nil
}

func (self *Model) View() string {
	return self.Outputs[self.TaskIndex]
}

var (
	ansiCursorSequence = regexp.MustCompile(`\x1B\[[0-9;]*[ABCD]`)
)

// ansiSanitize removes ANSI cursor sequences from a string, preserving the color codes.
func ansiSanitize(s string) string {
	return ansiCursorSequence.ReplaceAllString(s, "")
}

package main

import (
	"fmt"

	"github.com/fsnotify/fsnotify"
	"github.com/go-cmd/cmd"
	"github.com/k0kubun/pp/v3"
)

type TaskCmd int
type TaskStatus int

const (
	TaskCmdQuit TaskCmd = iota
	TaskCmdStart

	TaskRunning TaskStatus = iota
	TaskIdle
	TaskNewContents
	TaskFinished
	TaskError
	TaskRestarting
)

type Task struct {
	Prefix     []string
	Commands   []string
	MaxHistory uint64

	Output []byte

	Files []string

	SubtaskIndex int

	StatusLong string
	Status     TaskStatus

	Name string

	IsMenuTask bool
}

func (self *Task) Init() {
	self.Output = make([]byte, 0)
	self.Files = nil
	self.Status = TaskIdle
}

func (self *Task) Start(events <-chan TaskCmd) {
	// dont reset output

	var watcherEvt <-chan fsnotify.Event
	var watcherErr <-chan error

	if self.Files != nil {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			panic(err)
		}

		defer watcher.Close()

		for _, file := range self.Files {
			err = watcher.Add(file)
			if err != nil {
				panic(err)
			}
		}

		watcherEvt = watcher.Events
		watcherErr = watcher.Errors
	}

	streamingCmdOptions := cmd.Options{
		Buffered:  false,
		Streaming: true,
	}

	updateStatus := func(statusLong string, status TaskStatus) {
		self.StatusLong = statusLong
		self.Status = status
		self.appendOutput([]byte(statusLong))
	}

	wait := make(chan struct{})

	go func() {
		var index int
		if self.IsMenuTask {
			index = -1
		} else {
			index = 0
		}
		for {
			args := append(self.Prefix[1:], fmt.Sprintf("'%s'", self.Commands[index]))
			proc := cmd.NewCmdOptions(streamingCmdOptions, self.Prefix[0], args...)

			var cmdStatus <-chan cmd.Status

			if index >= 0 {
				self.SubtaskIndex = index

				cmdStatus = proc.Start()
				updateStatus(fmt.Sprintf("Running: %s\n", self.Commands[index]), TaskRunning)
			}

			select {
			case msg := <-events:
				switch msg {
				case TaskCmdQuit:
					proc.Stop()
					close(wait)
					return
				case TaskCmdStart:
					proc.Stop()
					index = 0 // start the menu task, or restart the current task
					continue
				}

			case msg := <-watcherEvt:
				status := fmt.Sprintf("Changed file: %s, Restarting\n", msg.Name)
				updateStatus(status, TaskRestarting)
				proc.Stop()
				index = 0
				continue

			case msg := <-watcherErr:
				if msg != nil {
					status := fmt.Sprintf("Watcher error: %s\n", msg)
					updateStatus(status, TaskError)
					proc.Stop()
					close(wait)
					return
				}

			case status := <-cmdStatus:
				if status.Complete {
					status := fmt.Sprintf("Finished: %s\n", status.Cmd)
					updateStatus(status, TaskFinished) // soon rewritten by next command
					index++
					if index == len(self.Commands) {
						close(wait)
						return
					}
				}

				if status.Error != nil || status.Exit != 0 {
					status := fmt.Sprintf("Error: %s\n", status.Error)
					updateStatus(status, TaskError)
					close(wait)
					return
				}

			case msg := <-proc.Stdout:
				self.appendOutput([]byte(msg))

			case msg := <-proc.Stderr:
				self.appendOutput([]byte(msg))
			}
		}
	}()

	<-wait
}

// Call atmost once at the start of setting this up
func (self *Task) Watch(files []string, exclude []string) {
	files, err := Globs(files)
	if err != nil {
		panic(err)
	}

	exclude, err = Globs(exclude)
	if err != nil {
		panic(err)
	}

	self.Files = SubtractSlice(files, exclude)

	pp.Println(self.Files)
}

func (self *Task) appendOutput(output []byte) {
	self.Output = append(self.Output, output...)
	if len(self.Output) > int(self.MaxHistory) {
		self.Output = self.Output[len(self.Output)-int(self.MaxHistory):]
	}
}

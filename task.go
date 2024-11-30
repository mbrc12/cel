package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/go-cmd/cmd"
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
	Format   string
	Commands []string
	LogFile  string

	OutputChan chan string

	Files []string

	SubtaskIndex int

	StatusLong string
	Status     TaskStatus

	Name string

	IsMenuTask bool

	Closed bool
}

func (self *Task) Init() {
	self.Files = nil
	self.Status = TaskIdle
	self.Closed = true
	self.SubtaskIndex = -1
}

func (self *Task) Start(events <-chan TaskCmd) {
	// dont reset output

	var err error

	self.Closed = false

	var logFile *os.File

	if self.LogFile != "" {
		logFile, err = os.Create(self.LogFile)
		if err != nil {
			panic(err)
		}
	}

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
		self.OutputChan <- fmt.Sprintf("%s\n", statusLong)
	}

	wait := make(chan struct{})

	go func() {
		proc := &cmd.Cmd{}

		nextTask := make(chan int, 1)

		if !self.IsMenuTask {
			nextTask <- 0
		}

		for {
			select {
			case index := <-nextTask:
				if self.SubtaskIndex >= 0 {
					proc.Stop() // stop previous task
				}

				fields := strings.Fields(self.Format)
				for i, field := range fields {
					fields[i] = strings.ReplaceAll(field, "%", self.Commands[index])
				}

				proc = cmd.NewCmdOptions(streamingCmdOptions, fields[0], fields[1:]...)
				proc.Start()

				self.SubtaskIndex = index

				updateStatus(fmt.Sprintf("Running: %s", self.Commands[index]), TaskRunning)

			case msg := <-events:
				switch msg {
				case TaskCmdQuit:
					if self.SubtaskIndex >= 0 {
						proc.Stop()
					}
					close(wait)
					return
				case TaskCmdStart:
					nextTask <- 0
					continue
				}

			case msg := <-watcherEvt:
				status := fmt.Sprintf("Changed file: %s, restarting ...", msg.Name)
				updateStatus(status, TaskRestarting)
				nextTask <- 0
				continue

			case msg := <-watcherErr:
				if msg != nil {
					status := fmt.Sprintf("Watcher error: %s", msg)
					updateStatus(status, TaskError)
					close(wait)
					return
				}

			case msg := <-proc.Stdout:
				msg = msg + "\n"
				if logFile != nil {
					_, err = logFile.WriteString(msg)
					if err != nil {
						panic(err)
					}
				}
				self.OutputChan <- msg

			case msg := <-proc.Stderr:
				if logFile != nil {
					_, err = logFile.WriteString(msg)
					if err != nil {
						panic(err)
					}
				}
				self.OutputChan <- msg

			case <-proc.Done():

				procStatus := proc.Status()

				if procStatus.Error != nil || procStatus.Exit != 0 {
					var status string

					if procStatus.Error == nil {
						status = fmt.Sprintf("Exited with code %d", procStatus.Exit)
					} else {
						status = fmt.Sprintf("Error: %s", procStatus.Error)
					}

					updateStatus(status, TaskError)
					close(wait)
					return
				}

				// finished successfully

				status := fmt.Sprintf("Finished.")
				updateStatus(status, TaskFinished) // soon rewritten by next command
				nextIndex := self.SubtaskIndex + 1
				if nextIndex == len(self.Commands) {
					self.SubtaskIndex = -1
					proc = &cmd.Cmd{}
					continue
				}
				nextTask <- nextIndex
			}
		}
	}()

	<-wait
	self.Closed = true
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
}

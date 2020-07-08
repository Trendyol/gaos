/*
Copyright 2020 The Gaos Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package logger

import (
	"fmt"
	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"os"
	"time"
)

var spinnerFrames = []string{
	"⠈⠁",
	"⠈⠑",
	"⠈⠱",
	"⠈⡱",
	"⢀⡱",
	"⢄⡱",
	"⢄⡱",
	"⢆⡱",
	"⢎⡱",
	"⢎⡰",
	"⢎⡠",
	"⢎⡀",
	"⢎⠁",
	"⠎⠁",
	"⠊⠁",
}

const (
	FATAL = "FATAL"
	ERROR = "ERROR"
	WARN  = "WARNING"
	INFO  = "INFO"
	DEBUG = "DEBUG"
	TRACE = "TRACE"
)

var log = NewLoggers()

type ILoggers interface {
	Debug(message interface{})
	Info(message interface{})
	Warn(message interface{})
	Error(message interface{})
	Fatal(message interface{})
	Trace(message interface{})
	Log(level string, message interface{})
	Spinner(message string) func()
}

type Loggers struct {
	spinner *spinner.Spinner
}

func NewLoggers() ILoggers {
	return &Loggers{
		spinner: spinner.New(spinnerFrames, 100*time.Millisecond),
	}
}

func Debug(message interface{}) {
	log.Log(DEBUG, message)
}

func (loggers *Loggers) Debug(message interface{}) {
	loggers.Log(DEBUG, message)
}

func Info(message interface{}) {
	log.Info(message)
}

func (loggers *Loggers) Info(message interface{}) {
	loggers.Log(INFO, message)
}

func Warn(message interface{}) {
	log.Warn(message)
}

func (loggers *Loggers) Warn(message interface{}) {
	loggers.Log(WARN, message)
}

func Error(message interface{}) {
	log.Error(message)
}

func (loggers *Loggers) Error(message interface{}) {
	loggers.Log(ERROR, message)
}

func Fatal(message interface{}) {
	log.Fatal(message)
	os.Exit(1)
}

func (loggers *Loggers) Fatal(message interface{}) {
	loggers.Log(FATAL, message)
}

func Trace(message interface{}) {
	log.Trace(message)
}

func (loggers *Loggers) Trace(message interface{}) {
	loggers.Log(TRACE, message)
}

func Log(level string, message interface{}) {
	log.Log(level,message)
}

func (loggers *Loggers) Log(level string, message interface{}) {
	loggers.spinner.Stop()

	c := color.New(color.FgWhite)

	if level == ERROR {
		c = color.New(color.FgRed, color.Bold)
	}

	if level == INFO {
		c = color.New(color.Italic)
	}

	_, _ = c.Println(fmt.Sprintf("⇨ %s", message))
}

func Spinner(message string) func() {
	return log.Spinner(message)
}

func (loggers *Loggers) Spinner(message string) func() {
	loggers.spinner.Suffix = " " + message
	_ = loggers.spinner.Color("fgGreen")
	loggers.spinner.Start()

	return loggers.spinner.Stop
}

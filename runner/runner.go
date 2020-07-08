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

package runner

import (
	"encoding/json"
	"fmt"
	"github.com/Trendyol/gaos/logger"
	"github.com/fasthttp/router"
	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/runtime"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const BANNER = `
╔══════════════════════════════════════════════════════════════════════════════════════╗
║ .................................................................................... ║
║ ........................... ╔════╗ ╔════╗ ╔════╗ ╔════╗ ............................ ║
║ ........................... ║ .. ║ ║ .. ║ ║ .. ║ ║ .. ║ ............................ ║
║ ........................... ║ .╔═╗ ╠════╣ ║ .. ║ ╚════╗ ............................ ║
║ ........................... ║ .. ║ ║ .. ║ ║ .. ║ ║ .. ║ ............................ ║
║ ........................... ╚════╝ ╩ .. ╩ ╚════╝ ╚════╝ ............................ ║
║ .................................................................................... ║
╚════════════════════════════════════ Trendyol Tech ═══════════════════════════════════╝
                                                                               (v%s)
`
const VERSION = "0.1.0"

const (
	ResultTypeStatic   = "static"
	ResultTypeFile     = "file"
	ResultTypeRedirect = "redirect"
)

const (
	FileResultTypeJson = "json"
)

type Executable func() (Done, error)

type Done <-chan bool

type Runner struct {
	Service  map[string]*Service  `json:"service"`
	Scenario map[string]*Scenario `json:"scenario"`
	servers  []*fasthttp.Server
}

type Service struct {
	Port int32           `json:"port"`
	Path map[string]Path `json:"path"`
}

type Path struct {
	Scenario string `json:"scenario"`
	Method   string `json:"method"`
}

type Action struct {
	scenario *Scenario
	Direct   string `json:"direct"`
	Status   int    `json:"status"`
	Result   Result `json:"result"`
}

type Result struct {
	Type    string      `json:"type"`
	Content interface{} `json:"content"`
}

type FileResult struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

type RedirectResult struct {
	Host string `json:"host"`
}

type Scenario struct {
	executables []Executable
	Name        string `json:"name"`
	Duration    string `json:"duration"`
	Latency     string `json:"latency"`
	Status      int    `json:"status"`
	Rate        int    `json:"rate"`
	Random      int    `json:"random"`
	Limit       int    `json:"limit"`
	Start       string `json:"start"`
	End         string `json:"end"`
	Accept      Action `json:"accept"`
	Ignore      Action `json:"ignore"`
}

type Method struct {
	Scenario
}

func New(path string) (*Runner, error) {
	runner := &Runner{}

	file, err := ioutil.ReadFile(path)

	if err != nil {
		return nil, errors.Wrap(err, "Unable to read scenario file")
	}

	err = json.Unmarshal(file, runner)

	if err != nil {
		return nil, errors.Wrap(err, "Unable to parse scenario file")
	}

	clr := color.New(color.FgMagenta)

	_, _ = clr.Println(fmt.Sprintf(BANNER, VERSION))

	return runner, nil
}

func (g *Runner) Run(services ...string) {

	g.resolveScenarios()

	result := true

	for name, service := range g.Service {

		if len(services) > 0 {
			f := false

			for _, s := range services {
				if s == name {
					f = true
					break
				}
			}

			if !f {
				continue
			}
		}

		result = result && g.runToService(service, name)
	}

	if len(g.servers) == 0 {
		logger.Fatal("There are no servers to run")
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	if !result {
		logger.Fatal("Something went wrong while running services")
	}

	logger.Info("Servers are stopping...")

	for _, v := range g.servers {
		err := v.Shutdown()

		if err != nil {
			logger.Error(fmt.Sprintf("[%s] Http server can not closed, %s", v.Name, err))
			continue
		}

		logger.Info(fmt.Sprintf("[%s] Http server closed", v.Name))
	}
}

func (g *Runner) runToService(service *Service, name string) bool {

	r := router.New()

	r.PanicHandler = func(ctx *fasthttp.RequestCtx, err interface{}) {
		g.ErrorHandler(ctx, errors.Errorf("%+v", err))
	}

	r.MethodNotAllowed = func(ctx *fasthttp.RequestCtx) {
		g.ErrorHandler(ctx, errors.New("Method not allowed"))
	}

	r.NotFound = func(ctx *fasthttp.RequestCtx) {
		g.ErrorHandler(ctx, errors.New("Not found"))
	}

	for path, value := range service.Path {

		if scenario, ok := g.Scenario[value.Scenario]; ok {

			method := Method{*scenario}

			r.Handle(value.Method, path, method.Handler())
		}

	}

	server := &fasthttp.Server{
		Name:    fmt.Sprint(service.Port),
		Handler: r.Handler,
		ErrorHandler: func(ctx *fasthttp.RequestCtx, err error) {
			g.ErrorHandler(ctx, err)
		},
	}

	result := true

	go func() {

		logger.Info(fmt.Sprintf("[%d] HTTP server started for [%s]", service.Port, name))

		err := server.ListenAndServe(fmt.Sprintf(":%d", service.Port))

		if err != nil {
			logger.Error(err)
		}

		result = result && (err == nil)
	}()

	g.servers = append(g.servers, server)

	return result
}

func (g *Runner) resolveScenarios() {

	for k := range g.Scenario {

		scenario := g.Scenario[k]

		if len(scenario.Start) > 0 || len(scenario.End) > 0 {

			span := NewSpan(*scenario)

			scenario.executables = append(scenario.executables, span.Execute)
		}

		if len(scenario.Duration) > 0 {

			duration := NewDuration(*scenario)

			scenario.executables = append(scenario.executables, duration.Execute)
		}

		if len(scenario.Latency) > 0 {

			latency := NewLatency(*scenario)

			scenario.executables = append(scenario.executables, latency.Execute)
		}

		if scenario.Limit > 0 {

			limit := NewLimit(*scenario)

			scenario.executables = append(scenario.executables, limit.Execute)
		}

		if scenario.Rate > 0 {

			rate := NewRate(*scenario)

			scenario.executables = append(scenario.executables, rate.Execute)
		}

		if len(scenario.Accept.Direct) > 0 {
			if v, ok := g.Scenario[scenario.Accept.Direct]; ok {
				scenario.Accept.scenario = v
			}
		}

		if len(scenario.Ignore.Direct) > 0 {
			if v, ok := g.Scenario[scenario.Ignore.Direct]; ok {
				scenario.Ignore.scenario = v
			}
		}
	}
}

func (g *Runner) ErrorHandler(ctx *fasthttp.RequestCtx, cause error) {

	e := WrapGaosError(cause, "Occurred http error")

	body, _ := json.Marshal(e)

	ctx.SetBody(body)
	ctx.SetContentType(runtime.ContentTypeJSON)
	ctx.SetStatusCode(fasthttp.StatusInternalServerError)
}

func (m *Method) Handler() fasthttp.RequestHandler {

	cnt := 0

	return func(ctx *fasthttp.RequestCtx) {

		start := time.Now()

		defer func(name string) {
			elapsed := time.Since(start) / time.Millisecond
			logger.Info(fmt.Sprintf("[%d] Host: %s | Path: %s | Executed: %s | Elapsed time: %dms", cnt, string(ctx.Host()), string(ctx.Request.URI().Path()), name, elapsed))
			cnt++
		}(m.Name)

		action, done := m.Execute()

		err := action.Execute(ctx)

		for _, d := range done {
			<-d
		}

		if err != nil {

			result := WrapGaosError(err, "Occurred a error")

			body, _ := json.Marshal(result)

			ctx.SetStatusCode(fasthttp.StatusInternalServerError)
			ctx.SetContentType(runtime.ContentTypeJSON)
			ctx.SetBody(body)

		}
	}
}

func (m *Method) Execute() (Action, []Done) {

	var done []Done

	e := m.executables
	action := m.Accept

	for _, method := range e {

		d, err := method()

		if d != nil {
			done = append(done, d)
		}

		if err != nil {
			action = m.Ignore
			logger.Error(err)
			break
		}
	}

	if action.scenario != nil {
		m.Scenario = *action.scenario
	}

	return action, done

}

func (a *Action) Execute(ctx *fasthttp.RequestCtx) error {

	if a.Result.Type == ResultTypeFile {

		if v, ok := a.Result.Content.(map[string]interface{}); ok {

			r := FileResult{
				Path: fmt.Sprint(v["path"]),
				Type: fmt.Sprint(v["type"]),
			}

			if r.Type == FileResultTypeJson {

				f, err := ioutil.ReadFile(r.Path)

				if err != nil {
					return errors.Wrap(err, "Result file can not read")
				}

				ctx.SetStatusCode(a.Status)
				ctx.SetBody(f)

				return nil
			}

		}
	} else if a.Result.Type == ResultTypeRedirect {

		if v, ok := a.Result.Content.(map[string]interface{}); ok {

			r := RedirectResult{
				Host: fmt.Sprint(v["host"]),
			}

			req := fasthttp.AcquireRequest()
			res := fasthttp.AcquireResponse()

			defer fasthttp.ReleaseResponse(res)
			defer fasthttp.ReleaseRequest(req)

			ctx.Request.CopyTo(req)

			req.SetRequestURI(r.Host + string(ctx.Path()))

			err := fasthttp.Do(req, res)

			if err != nil {
				return errors.Wrap(err, "HTTP request can not send")
			}

			res.CopyTo(&ctx.Response)

			return nil

		}

		return nil

	} else if a.Result.Type == ResultTypeStatic {

		body, err := json.Marshal(a.Result.Content)

		if err != nil {
			return errors.Wrap(err, "Result content marshalling error")
		}

		ctx.SetStatusCode(a.Status)
		ctx.SetContentType(runtime.ContentTypeJSON)
		ctx.SetBody(body)

		return nil
	}

	ctx.SetStatusCode(fasthttp.StatusNoContent)
	ctx.SetContentType(runtime.ContentTypeJSON)

	return nil
}

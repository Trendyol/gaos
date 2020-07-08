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
	"github.com/pkg/errors"
	"time"
)

type Limit struct {
	s Scenario
	l int
	n int
}

func NewLimit(s Scenario) *Limit {
	return &Limit{
		s: s,
		l: s.Limit,
		n: 1,
	}
}

func (l *Limit) Execute() (Done, error) {

	l.n++

	if l.n > l.l {
		return nil, errors.New("Request count exceed scenario limit")
	}

	return nil, nil
}

type Rate struct {
	s Scenario
	r int
	n int
}

func NewRate(s Scenario) *Rate {
	return &Rate{
		s: s,
		r: s.Rate,
		n: 1,
	}
}

func (r *Rate) Execute() (Done, error) {

	r.n++

	if r.n > r.r {
		r.n = 0
		return nil, errors.New("Request count exceed scenario rate limit")
	}

	return nil, nil
}

type Duration struct {
	s        Scenario
	duration time.Duration
}

func NewDuration(s Scenario) *Duration {
	duration, _ := time.ParseDuration(s.Duration)
	return &Duration{
		s:        s,
		duration: duration,
	}
}

func (d *Duration) Execute() (Done, error) {
	done := make(chan bool)

	timer := time.NewTimer(d.duration)

	go func(c <-chan time.Time, d chan bool) {
		<-c
		d <- true
	}(timer.C, done)

	return done, nil
}

type Latency struct {
	s     Scenario
	sleep time.Duration
}

func NewLatency(s Scenario) *Latency {
	sleep, _ := time.ParseDuration(s.Latency)
	return &Latency{
		s:     s,
		sleep: sleep,
	}
}

func (d *Latency) Execute() (Done, error) {

	time.Sleep(d.sleep)

	return nil, nil
}

type Span struct {
	s     Scenario
	start *time.Time
	end   *time.Time
}

func NewSpan(s Scenario) *Span {
	start, serr := time.Parse("2006-01-02T15:04:05.999999Z", s.Start)
	end, eerr := time.Parse("2006-01-02T15:04:05.999999Z", s.End)

	span := &Span{
		s:     s,
		start: &start,
		end:   &end,
	}

	if len(s.Start) == 0 || serr != nil {
		span.start = nil
	}

	if len(s.Start) == 0 || eerr != nil {
		span.end = nil
	}

	return span
}

func (d *Span) Execute() (Done, error) {

	n := time.Now()

	if d.start != nil && d.start.After(n) {
		return nil, errors.New(n.Format("2006-01-02T15:04:05.999999Z") + " this time is not between to scenario start and end time")
	}

	if d.end != nil && d.end.Before(n) {
		return nil, errors.New(n.Format("2006-01-02T15:04:05.999999Z") + " this time is not between to scenario start and end time")
	}

	return nil, nil
}

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

package executor

import (
	"github.com/pkg/errors"
	"zeus-gitlab.trendyol.com/general/gaos/runner"
)

const DOCKER = "docker"
const K8S = "k8s"

type Executor interface {
	Initialize(c Config) error
	Run() error
}

func NewExecutor(config Config) (Executor, error) {

	gaos, err := runner.New(config.Scenario)

	if err != nil {
		return nil, err
	}

	if config.Environment == DOCKER {

		docker, err := NewDocker(*gaos)

		if err != nil {
			return nil, err
		}

		err = docker.Initialize(config)

		if err != nil {
			return nil, err
		}

		return docker, nil

	} else if config.Environment == K8S {

		k8s, err := NewKubernetes(*gaos)

		if err != nil {
			return nil, err
		}

		err = k8s.Initialize(config)

		if err != nil {
			return nil, err
		}

		return k8s, nil

	}

	return nil, errors.New("Unexpected environment given. Available: 'docker', 'k8s'")
}

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
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/Trendyol/gaos/logger"
	"github.com/Trendyol/gaos/runner"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/jhoonb/archivex"
	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

const dockerfile = `FROM trendyol/gaos:%s

ADD ./scenario.json .

EXPOSE %d  

ENTRYPOINT ["./gaos", "run", "-x", "%s"]`

var timeout = 20 * time.Second

type Docker struct {
	runner            runner.Runner
	client            *client.Client
	scenario          string
	registry          string
	username          string
	password          string
	timeout           string
	continueOnFailure string
}

func NewDocker(g runner.Runner) (*Docker, error) {

	scenarioJson, err := json.Marshal(g)

	if err != nil {
		return nil, err
	}

	docker, err := client.NewEnvClient()

	if err != nil {
		return nil, errors.Wrap(err, "Docker -> Unable to init new docker client env")
	}

	engine := &Docker{
		runner:   g,
		client:   docker,
		scenario: string(scenarioJson),
	}

	return engine, nil
}

func (d *Docker) Initialize(c Config) error {

	config := c.GetMyConfig("docker")

	d.registry = config["Registry"]
	d.username = config["Username"]
	d.password = config["Password"]
	d.timeout = config["Timeout"]
	d.continueOnFailure = config["ContinueOnFailure"]

	if t, err := time.ParseDuration(d.timeout); err == nil {
		timeout = t
	} else {
		return errors.Wrapf(err, "Docker -> Timeout value can not parsed. Value: %s", d.timeout)
	}

	return nil
}

func (d *Docker) Run() error {

	var ids = make([]string, 0, len(d.runner.Service))

	defer func() {
		for _, id := range ids {

			err := d.stopContainer(id)

			if err != nil {
				logger.Trace(errors.Wrap(err, "Docker -> Service can not stopping"))
			}
		}
	}()

	for name, service := range d.runner.Service {

		image, err := d.createImage(name, service.Port)

		if err != nil {
			if len(ids) > 0 && !d.ask(err.Error()+" Do you want to stop all running container?") {
				continue
			}

			return errors.Wrap(err, "Docker -> Service can not running")
		}

		id, err := d.startContainer(*image)

		if err != nil {
			if len(ids) > 0 && !d.ask(err.Error()+" Do you want to stop all running container?") {
				continue
			}

			return errors.Wrap(err, "Docker -> Service can not running")
		}

		ids = append(ids, id)
	}

	d.prompt()

	return nil
}

func (d *Docker) createTar(name string, port int32) (string, error) {

	dir, err := ioutil.TempDir(".", "tmp-docker")

	defer func() {
		removeDirErr := os.RemoveAll(dir)

		if removeDirErr != nil {
			logger.Trace(fmt.Sprintf("Docker -> Temp directory can not removed. Error: %+v", removeDirErr))
		}
	}()

	if err != nil {
		return "", errors.Wrap(err, "Docker -> Temp directory can not created")
	}

	_dockerfile := fmt.Sprintf(dockerfile, runner.VERSION, port, name)

	err = ioutil.WriteFile(fmt.Sprintf("%s/%s", dir, "Dockerfile"), []byte(_dockerfile), 0644)

	if err != nil {
		return "", errors.Wrapf(err, "Docker -> Temp Dockerfile can not created at dir: %s", dir)

	}

	err = ioutil.WriteFile(fmt.Sprintf("%s/%s", dir, "scenario.json"), []byte(d.scenario), 0644)

	if err != nil {
		return "", errors.Wrap(err, "Docker -> Temp scenario.json can not created")
	}

	tarName := fmt.Sprintf("%s.tar", dir)

	tar := new(archivex.TarFile)
	err = tar.Create(tarName)

	if err != nil {
		return "", errors.Wrapf(err, "Docker -> Tar file can not created: %s", tarName)
	}

	err = tar.AddAll(dir, false)

	if err != nil {
		return "", errors.Wrapf(err, "Docker -> Unable to add all Tar files to given dir: %s", dir)
	}

	err = tar.Close()

	if err != nil {
		return "", errors.Wrap(err, "Docker -> Unable to close Tar file")
	}

	return tarName, nil
}

func (d *Docker) removeTar(tarName string) {
	removeTarErr := os.Remove(tarName)

	if removeTarErr != nil {
		logger.Trace(fmt.Sprintf("Docker -> Tar file can not removed. Error: %+v", removeTarErr))
	}
}

func (d *Docker) createImage(name string, port int32) (*Image, error) {
	defer logger.Spinner(fmt.Sprintf("Docker -> Creating image: [%s]", name))()

	if len(d.registry) > 0 && !strings.HasSuffix(d.registry, "/") {
		d.registry += "/"
	}

	id := fmt.Sprintf("%s-%s", name, time.Now().Format("20060102150405"))

	image := fmt.Sprintf("%s%s:%s", d.registry, "gaos", id)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	defer cancel()

	tarName, err := d.createTar(name, port)

	if err != nil {
		return nil, errors.Wrapf(err, "Docker -> Unable to create Tar file with name: %s and port: %d", name, port)
	}

	defer d.removeTar(tarName)

	dockerCtx, err := os.Open(tarName)

	if err != nil {
		return nil, errors.Wrapf(err, "Docker -> Unable to open Tar file: %s", tarName)
	}

	options := types.ImageBuildOptions{
		Tags: []string{image},
	}

	body, err := d.client.ImageBuild(ctx, dockerCtx, options)

	if err != nil {
		return nil, errors.Wrap(err, "Docker -> Unable to build image")
	}

	bodyReader := bufio.NewReader(body.Body)

	defer func() {
		err := body.Body.Close()

		if err != nil {
			logger.Trace("Docker -> Unable to close ReadCloser body")
		}
	}()

	for {
		streamBytes, err := bodyReader.ReadBytes('\n')

		if err == io.EOF {
			break
		}

		progress := map[string]interface{}{}

		errorBodyErr := json.Unmarshal(streamBytes, &progress)

		if errorBodyErr != nil {
			return nil, errors.Errorf("Docker -> Unable to push image to registry. Cause: %s", errorBodyErr)
		}

		if v, ok := progress["error"]; ok && v != nil && len(v.(string)) > 0 {
			return nil, errors.Errorf("Docker -> Unable to push image to registry. Cause: %s", v)
		}
	}

	logger.Debug(fmt.Sprintf("Docker -> Image created. Image: %s", image))

	if len(d.registry) > 0 {
		err = d.pushImage(image)

		if err != nil {
			return nil, err
		}
	}

	result := &Image{
		Id:       id,
		Title:    name,
		Name:     image,
		Port:     port,
		Registry: d.registry,
	}

	return result, nil
}

func (d *Docker) pushImage(image string) error {
	defer logger.Spinner(fmt.Sprintf("Docker -> Image is pushing. Image: %s", image))()

	docker, err := client.NewEnvClient()

	if err != nil {
		return errors.Wrap(err, "Docker -> Unable to create client environment")
	}

	authConfig := types.AuthConfig{
		Username: d.username,
		Password: d.password,
	}

	encodedJSON, _ := json.Marshal(authConfig)

	authStr := base64.URLEncoding.EncodeToString(encodedJSON)

	options := types.ImagePushOptions{
		RegistryAuth: authStr,
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	defer cancel()

	body, err := docker.ImagePush(ctx, image, options)

	if err != nil {
		return errors.Wrapf(err, "Docker -> Unable to push image to registry. Image: %s", image)
	}

	bodyReader := bufio.NewReader(body)

	defer func() {
		err := body.Close()

		if err != nil {
			logger.Trace("Docker -> Unable to close ReadCloser body")
		}
	}()

	for {
		streamBytes, err := bodyReader.ReadBytes('\n')

		if err == io.EOF {
			break
		}

		progress := map[string]interface{}{}

		errorBodyErr := json.Unmarshal(streamBytes, &progress)

		if errorBodyErr != nil {
			return errors.Wrap(errorBodyErr, "Docker -> Unable to push image to registry.")
		}

		if v, ok := progress["error"]; ok && v != nil && len(v.(string)) > 0 {
			return errors.Errorf("Docker -> Unable to push image to registry. Cause: %s", v)
		}
	}

	logger.Debug(fmt.Sprintf("Docker -> Image is successful push to registry. Image: %s", image))

	return nil
}

func (d *Docker) startContainer(image Image) (string, error) {
	defer logger.Spinner(fmt.Sprintf("Docker -> Container is starting. Image: %s", image.Name))()

	var id string

	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	defer cancel()

	port := nat.Port(fmt.Sprint(image.Port))

	containerConfig := &container.Config{
		Image: image.Name,
		ExposedPorts: nat.PortSet{
			port: struct{}{},
		},
	}

	hostConfig := &container.HostConfig{
		PortBindings: map[nat.Port][]nat.PortBinding{
			port: {
				{
					HostIP:   "127.0.0.1",
					HostPort: fmt.Sprint(image.Port),
				},
			},
		},
	}

	cnt, err := d.client.ContainerCreate(ctx, containerConfig, hostConfig, nil, image.Id)

	if err != nil {
		return id, errors.Wrap(err, "Docker -> Unable to create container")
	}

	err = d.client.ContainerStart(ctx, cnt.ID, types.ContainerStartOptions{})

	if err != nil {
		return id, errors.Wrap(err, "Docker -> Unable to start container")
	}

	id = cnt.ID

	logger.Debug(fmt.Sprintf("Docker -> Container is running. ID: %s", id))

	return id, nil
}

func (d *Docker) stopContainer(id string) error {
	defer logger.Spinner(fmt.Sprintf("Docker -> Container is stopping, ID: %s", id))()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	docker, err := client.NewEnvClient()

	if err != nil {
		cancel()
		return errors.Wrapf(err, "Docker -> Unable to create client environment for id: %s", id)
	}

	err = docker.ContainerStop(ctx, id, nil)

	if err != nil {
		cancel()
		return errors.Wrapf(err, "Docker -> Unable to stop container for id: %s", id)
	}

	logger.Debug(fmt.Sprintf("Docker ->  Container is stopped. ID: %s", id))

	cancel()

	return nil
}

func (d *Docker) prompt() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
}

func (d *Docker) ask(message string) bool {

	switch d.continueOnFailure {
	case "1":
		return true
	case "2":
		return false
	}

	prompt := promptui.Select{
		Label: message + " [Y/N]",
		Items: []string{"Y", "N"},
	}
	_, result, err := prompt.Run()
	if err != nil {
		return true
	}

	return result == "Y"
}

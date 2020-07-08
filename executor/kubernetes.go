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
	"encoding/json"
	"flag"
	"fmt"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
	"strconv"
	"zeus-gitlab.trendyol.com/general/gaos/logger"
	"zeus-gitlab.trendyol.com/general/gaos/runner"
)

type Kubernetes struct {
	runner    runner.Runner
	Scenario  string
	Namespace string
	Usage     string
	Replica   int32
	CPU       string
	Memory    string
	Secret    string
	client    *kubernetes.Clientset
	docker    *Docker
}

func NewKubernetes(g runner.Runner) (*Kubernetes, error) {

	scenarioJson, err := json.Marshal(g)

	if err != nil {
		return nil, err
	}

	docker, err := NewDocker(g)

	if err != nil {
		return nil, errors.Wrap(err, "K8S -> Kubernetes agent can not started")
	}
	engine := &Kubernetes{
		runner:   g,
		docker:   docker,
		Scenario: string(scenarioJson),
	}

	return engine, nil
}

func (k *Kubernetes) Initialize(c Config) error {
	config := c.GetMyConfig("k8s")

	replica, err := strconv.ParseInt(config["Replica"], 10, 32)

	if err != nil {
		replica = 1
	}

	k.Scenario = config["Scenario"]
	k.Namespace = config["Namespace"]
	k.Usage = config["Usage"]
	k.Replica = int32(replica)
	k.CPU = config["Cpu"]
	k.Memory = config["Memory"]
	k.Secret = config["Secret"]
	k.Usage = config["Config"]

	k8sConfigStr := flag.String("kubeconfig", filepath.Join(homedir.HomeDir(), ".kube", "config"), k.Usage)

	flag.Parse()

	k8sConfig, err := clientcmd.BuildConfigFromFlags("", *k8sConfigStr)

	if err != nil {
		return errors.Wrap(err, "K8S -> Unable to get config from flags")
	}

	k.client, err = kubernetes.NewForConfig(k8sConfig)

	if err != nil {
		return errors.Wrapf(err, "K8S -> Unable to initialize new config at host: %s", k8sConfig.Impersonate)
	}

	err = k.docker.Initialize(c)

	if err != nil {
		return errors.Wrapf(err, "K8S -> Unable to initialize Docker at registry: %s", c.Registry)
	}

	return nil
}

func (k *Kubernetes) Run() error {

	for name, service := range k.runner.Service {

		image, err := k.docker.createImage(name, service.Port)

		if err != nil {
			return errors.Wrap(err, "K8S -> Service can not running")
		}

		err = k.createDeployment(*image)

		if err != nil {
			return errors.Wrap(err, "K8S -> Service can not running")
		}

		err = k.createService(*image)

		if err != nil {
			return errors.Wrap(err, "K8S -> Service can not running")
		}

	}

	return nil
}

func (k *Kubernetes) createDeployment(image Image) error {
	defer logger.Spinner(fmt.Sprintf("K8S -> Deployment is creating. Image: %s", image.Name))()

	deploymentClient := k.client.AppsV1().Deployments(k.Namespace)

	deployment, err := deploymentClient.Get(fmt.Sprintf("%s-deployment", image.Title), metav1.GetOptions{})

	deploymentSpec := apiv1.PodSpec{
		Containers: []apiv1.Container{
			{
				Name:  fmt.Sprintf("%s-api", image.Title),
				Image: image.Name,
				Ports: []apiv1.ContainerPort{
					{
						Name:          "http",
						Protocol:      apiv1.ProtocolTCP,
						ContainerPort: image.Port,
					},
				},
				Resources: apiv1.ResourceRequirements{
					Limits: apiv1.ResourceList{
						"cpu":    resource.MustParse(k.CPU),
						"memory": resource.MustParse(k.Memory),
					},
					Requests: apiv1.ResourceList{
						"cpu":    resource.MustParse(k.CPU),
						"memory": resource.MustParse(k.Memory),
					},
				},
			},
		},
	}

	if len(k.Secret) > 0 {
		deploymentSpec.ImagePullSecrets = []apiv1.LocalObjectReference{
			{
				Name: k.Secret,
			},
		}
	} else {
		deploymentSpec.Containers[0].ImagePullPolicy = "IfNotPresent"
	}

	if err == nil && deployment != nil {

		deployment.Spec.Replicas = int32Ptr(k.Replica)
		deployment.Spec.Template.Spec = deploymentSpec

		_, err := deploymentClient.Update(deployment)

		if err != nil {
			return errors.Wrapf(err, "K8S -> Unable to update deployment: %s", deployment.Name)
		}

	} else {

		deployment = &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("%s-deployment", image.Title),
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: int32Ptr(k.Replica),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": image.Title,
					},
				},
				Template: apiv1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": image.Title,
						},
					},
					Spec: deploymentSpec,
				},
			},
		}

		_, err := deploymentClient.Create(deployment)

		if err != nil {
			return errors.Wrapf(err, "K8S -> Unable to create deployment: %s", deployment.Name)
		}
	}

	logger.Info(fmt.Sprintf("K8S -> Deployment [%s] created on namespace [%s]", deployment.Name, deployment.Namespace))

	return nil
}

func (k *Kubernetes) createService(image Image) error {
	defer logger.Spinner(fmt.Sprintf("K8S -> Service is creating. Image: %s", image.Name))()

	serviceClient := k.client.CoreV1().Services(k.Namespace)

	service, err := serviceClient.Get(fmt.Sprintf("%s-service", image.Title), metav1.GetOptions{})

	servicePorts := []apiv1.ServicePort{
		{
			Port:       80,
			TargetPort: intstr.FromInt(int(image.Port)),
		},
	}

	if err == nil && service != nil {

		service.Spec.Ports = servicePorts

		service, err = serviceClient.Update(service)

		if err != nil {
			return errors.Wrapf(err, "K8S -> Unable to update service: %s", service.Name)
		}

	} else {

		service := &apiv1.Service{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Service",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-service", image.Title),
				Namespace: k.Namespace,
			},
			Spec: apiv1.ServiceSpec{
				Ports: servicePorts,
				Selector: map[string]string{
					"app": image.Title,
				},
			},
		}

		service, err = serviceClient.Create(service)

		if err != nil {
			return errors.Wrapf(err, "K8S -> Unable to create service: %s-service", image.Title)
		}

	}

	logger.Debug(fmt.Sprintf("K8S -> Service [%s] created on port [%d]", service.Name, image.Port))

	logger.Debug(fmt.Sprintf("K8S -> Service information: `kubectl describe services %s --namespace %s`", service.Name, service.Namespace))

	return nil
}

func int32Ptr(i int32) *int32 { return &i }

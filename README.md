<p align="center"><a href="https://github.com/Trendyol/gaos" target="_blank"><img height="128" src="https://raw.githubusercontent.com/Trendyol/gaos/master/.res/logo.png"></a></p>

<h1 align="center">GAOS</h1>

<div align="center">
 <strong>
   HTTP mocking to test API services for chaos scenarios
 </strong>
</div>

<br />

<p align="center">
  <a href="https://github.com/ellerbrock/open-source-badges/"><img src="https://badges.frapsoft.com/os/v1/open-source.png?v=103" alt="Open Source"></a>
  <a href="https://opensource.org/licenses/Apache-2.0"><img src="https://img.shields.io/badge/License-Apache%202.0-blue.svg" alt="Apache 2.0"></a>
  <a href="https://goreportcard.com/badge/github.com/Trendyol/gaos"><img src="https://goreportcard.com/report/github.com/Trendyol/gaos" alt="Go Report"></a>
  <a href="https://github.com/Trendyol/gaos/workflows/test/badge.svg"><img src="https://github.com/Trendyol/gaos/actions?query=workflow%3Atest" alt="Build Status"></a>
  <a href="https://img.shields.io/github/tag/Trendyol/gaos.svg"><img src="https://github.com/Trendyol/gaos/releases/latest" alt="Tag"></a>
</p>

<br />

*Gaos*, can create and provide custom mock restful services via using your fully-customizable scenarios and runs them on Docker & Kubernetes & localhost flawlessly.

**Warning:** Currently in Beta.

## Features

> * API response mocking
> * Custom behaviour scenarios
> * Create custom actions for each scenario
> * Robust routing
> * Serve static & dynamic responses
> * Duration, Latency, Error scenarios
> * Deploy your services on Docker & K8S
> * ... and much more, explore the Gaos!

## Installation

* Via HomeBrew
```bash
$ brew tap trendyol/trendyol-tap
$ brew install gaos
```

* Via Go
```bash
$ go get -u zeus-gitlab.trendyol.com/general/gaos 
```

* Build on Docker
```bash
$ make build
```

## Scenarios

| Scenario		         | Explanation								      |
| ---------------------- |:----------------------------------------------:|
| `latency`				 | Adds extra latency for request  |
| `duration`		     | Adds duration limit for request  |
| `span`				 | Executes `accept` if in the specified time range, `ignore` otherwise.  |
| `rate`				 | Executes `ignore` if reaches the value or on multiples of, `accept` otherwise.  |

## Actions

| Action		         | Explanation								      |
| ---------------------- |:----------------------------------------------:|
| `accept`				 | Execute if span and rate conditions doesn't match in the specified scenario  |
| `ignore`				 | Execute if span and rate conditions does match in the specified scenario  |
| `direct`				 | Specifies which scenario should be handled by next request  |
| `result`				 | Specifies *Result Type* that will be return eventually  |
| `status`				 | Specifies *Status Code* for given *Result Type*  |

## Results

| Result		         | Explanation								      |
| ---------------------- |:----------------------------------------------:|
| `type`				 | *Result Type* of the content  |
| `content`				 | Definitions the relevant result's content, according to *Result Type*  |

### Result Type

| Result Type		     | Explanation								      |
| ---------------------- |:----------------------------------------------:|
| `static`				 | Returns `json` content  |
| `file`				 | Returns `content.type` after reading the file in `content.path`  |
| `redirect`		     | Sends request to remote `content.host` and awaits response _(reverse proxy)_ |

## Usage

```bash
Usage:
  gaos [command]

Available Commands:
  help        Help about any command
  run         Run Gaos server on localhost
  start       Start Gaos server on given engine (Docker, K8S)

Flags:
  -h, --help   help for gaos
```

### Run Command

```bash
Run Gaos server on your localhost

Usage:
  gaos run [flags]

Flags:
  -x, --execute string    execute scenario services
  -s, --scenario string   scenario file input (default "./scenario.json")
```

Example:

```bash
$ gaos run -s ./examples/scenario.json
```

After Gaos server started:

```bash
$ go run ./examples/example.go
```

### Start Command

```bash
Start Gaos server on given engine (Docker, K8S)

Usage:
  gaos start [flags]

Flags:
  -c, --config string        choose k8s config (default "minikube")
      --cpu string           cpu limit (default "500m")
  -e, --environment string   gaos running environment {docker, k8s} (default "local")
	  --memory string        memory limit (default "500mi")
  -n, --namespace string     choose namespace (default "default")
  -p, --password string      image registry password
  -r, --registry string      image registry
	  --replica string       replica count (default "1")
  -s, --scenario string      scenario file input (default "./scenario.json")
	  --secret string        secret key name
  -t, --timeout string       client timeout (default "5m")
  -u, --username string      image registry username
```

* Example: Docker

```bash
$ gaos start -e docker -s './examples/scenario.json)'
```

* Example: Kubernetes

```bash
$ gaos start -e k8s -s './examples/scenario.json'
```

## Running Tests

*Requirements:*

* [bats](https://github.com/bats-core/bats-core)
* [docker](https://www.docker.com/)
* [kind](https://github.com/kubernetes-sigs/kind) (with `kind-kind` context, `localhost:5000` [local registry](https://kind.sigs.k8s.io/docs/user/local-registry/))

```bash
$ bats e2e.bats
```

## Known Issues

* Lack of some unit tests
* E2E tests are not running on pipeline yet. (Manually: `bats e2e.bats`)
* Not tested in Windows yet

## TO-DO

* [ ] Add `./docs` folder for better documentation
* [ ] Scenario linter - to check rules, keys and paths
* [ ] Remote server config reader
* [ ] API client
* [ ] [Envoy](https://www.envoyproxy.io/) support - sidecar feature
* [ ] [Consul](https://www.consul.io/) support - service mesh feature

## License

The base project code is licensed under Apache License unless otherwise specified. Please see the **[LICENSE](https://github.com/Trendyol/gaos/blob/master/LICENSE)** file for more information.

<kbd>GAOS</kbd>

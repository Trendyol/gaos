#!/usr/bin/env bats

COMMAND="${COMMAND:/-$BATS_TEST_DIRNAME}"

: ${CMD:=$GOPATH/bin/gaos}

setup() {
    TEST_SCENARIO_PASS='./examples/scenario.json'
    TEST_SCENARIO_FAIL='/dev/null'
    TEST_EXECUTES_PASS='product,search'
    TEST_EXECUTES_FAIL='product;search'
    TEST_REGISTRY='localhost:5000'
    TEST_ENV_BAD='some'
    TEST_ENV_D4R='docker'
    TEST_ENV_K8S='k8s'
}

@test "project: clear" {
    run rm -rf "$CMD"
    echo "$output"
    [ "$status" -eq 0 ]
}

@test "project: should build" {
    run go build -o "$CMD" .
    echo "$output"
    [ "$status" -eq 0 ]
}


@test "project: run: should run" {
	run ${CMD}
	echo "status = ${status}">&2
	echo "output = ${output}">&2
	[ "$status" -eq 0 ]
	[[ $output = *"" ]]
}

@test "project: run: should not run without scenario" {
	run ${CMD} run
	echo "status = ${status}">&2
	echo "output = ${output}">&2
	[ "$status" -eq 1 ]
	[[ $output = *"Unable to read scenario file"* ]]
}

@test "project: run: should not run with non exist scenario" {
	run timeout --preserve-status 5 ${CMD} run -s $TEST_SCENARIO_FAIL
	echo "status = ${status}">&2
	[ "$status" -eq 1 ]
}

@test "project: run: should run scenario" {
	run timeout --preserve-status 5 ${CMD} run -s $TEST_SCENARIO_PASS
	echo "status = ${status}">&2
	echo "output = ${output}">&2
	[ "$status" -eq 0 ]
	[[ $output = *"HTTP server started for [product]"* ]]
	[[ $output = *"Servers are stopping..."* ]]
	[[ $output = *"Http server closed"* ]]
}

@test "project: run: should not run scenario with bad custom executes" {
	run timeout --preserve-status 5 ${CMD} run -s $TEST_SCENARIO_PASS -x $TEST_EXECUTES_FAIL
	echo "status = ${status}">&2
	echo "output = ${output}">&2
	[ "$status" -eq 1 ]
	[[ $output = *"There are no servers to run"* ]]
}

@test "project: run: should run with scenario and custom executes" {
	run timeout --preserve-status 5 ${CMD} run -s $TEST_SCENARIO_PASS -x $TEST_EXECUTES_PASS
	echo "status = ${status}">&2
	echo "output = ${output}">&2
	[ "$status" -eq 0 ]
	[[ $output = *"[9080] HTTP server started for [product]"* ]]
	[[ $output = *"[9081] HTTP server started for [search]"* ]]
	[[ $output = *"Servers are stopping..."* ]]
	[[ $output = *"[9080] Http server closed"* ]]
	[[ $output = *"[9081] Http server closed"* ]]
}

@test "project: start: should not start without scenario" {
	run ${CMD} start
	echo "status = ${status}">&2
	echo "output = ${output}">&2
	[ "$status" -eq 1 ]
	[[ $output = *"Unable to read scenario file"* ]]
}

@test "project: start: should not start without environment" {
	run ${CMD} start -s $TEST_SCENARIO_PASS
	echo "status = ${status}">&2
	echo "output = ${output}">&2
	[ "$status" -eq 1 ]
	[[ $output = *"Unexpected environment given"* ]]
}

@test "project: start: check prerequisites" {
    run command -v kind
    [ "$status" -eq 0 ]
    run command -v kubectl
    [ "$status" -eq 0 ]
    run command -v docker
    [ "$status" -eq 0 ]
    run docker ps
    [ "$status" -eq 0 ]
    run kubectl config use-context kind-kind
    [ "$status" -eq 0 ]
}

@test "project: start: docker: should start deployment and push to docker" {
	run timeout --signal INT --preserve-status 150 ${CMD} start -s $TEST_SCENARIO_PASS -e $TEST_ENV_D4R -r $TEST_REGISTRY
	echo "status = ${status}">&2
	echo "output = ${output}">&2
	[ "$status" -eq 0 ]
	[[ $output = *"Docker: Creating image: [product]"* ]]
	[[ $output = *"Container is running."* ]]
	[[ $output = *"Docker: Creating image: [search]"* ]]
	[[ $output = *"Container is running."* ]]
	[[ $output = *"Containers are stopping..."* ]]
	[[ $output = *"Docker container is stopped"* ]]
	[[ $output = *"Docker container is stopped"* ]]
}

@test "project: start: k8s: should start deployment and push to kind using registry" {
	run ${CMD} start -s $TEST_SCENARIO_PASS -e $TEST_ENV_K8S -r $TEST_REGISTRY -n default --replica 1
	echo "status = ${status}">&2
	echo "output = ${output}">&2
	[ "$status" -eq 0 ]
	[[ $output = *"Docker: Creating image: [product]"* ]]
	[[ $output = *"Service [product-service] created on port [9080]"* ]]
	[[ $output = *"Docker: Creating image: [search]"* ]]
	[[ $output = *"Service [search-service] created on port [9081]"* ]]
}

@test "project: start: k8s: check false-positives" {
	run kubectl get deployment product-deployment -n default
	[ "$status" -eq 0 ]
    run kubectl get deployment search-deployment -n default
	[ "$status" -eq 0 ]

	run kubectl get service product-service -n default
    [ "$status" -eq 0 ]
	run kubectl get service search-service -n default
	[ "$status" -eq 0 ]

	run kubectl get pods --selector=app=product
	[ "$status" -eq 0 ]
	[[ ! $output = *"No resources found."* ]]

	run kubectl get pods --selector=app=search
    [ "$status" -eq 0 ]
    [[ ! $output = *"No resources found."* ]]
}
.PHONY: build run

build:
	docker build -t gaos \
		--build-arg VERSION=`git describe --abbrev=0 --tag` \
		--build-arg COMMIT=`git rev-parse --short HEAD` \
		--build-arg DATE=`date -u +"%Y-%m-%dT%H:%M:%SZ"` \
		.

run:
	docker exec -i -t gaos:latest /bin/bash
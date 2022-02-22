build:
	go build -o bin/check_docker_swarm src/check_docker_swarm.go

build-static:
	go build -ldflags="-extldflags=-static" -tags 'osusergo netgo' -o bin/check_docker_swarm src/check_docker_swarm.go

clean:
	go clean
	rm -rf bin/

all: clean build

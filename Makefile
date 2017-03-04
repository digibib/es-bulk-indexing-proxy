build:
	CGO_ENABLED=0 go build

TAG=$(shell git rev-parse HEAD)
docker: build
	docker build -t=digibib/es-bulk-indexing-proxy:$(TAG) .
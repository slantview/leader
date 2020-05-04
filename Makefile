VERSION := 0.1.0
GIT_HASH := $(shell git rev-parse --short HEAD)
BIN := leader

build:
	go build -o $(BIN) .

clean:
	rm -rf bin/linux/$(BIN) $(BIN)

bin/linux/$(BIN):
	GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION) -X main.hash=$(GIT_HASH)" -o bin/linux/$(BIN) .

docker: clean bin/linux/$(BIN)
	docker build -t slantview/$(BIN):$(VERSION)-$(GIT_HASH) .

docker-run: docker
	docker run -it --net host slantview/$(BIN):$(VERSION)-$(GIT_HASH)

start-consul: 
	consul agent -dev &

start-etcd:
	etcd &

.PHONY: build clean docker start-consul start-etcd

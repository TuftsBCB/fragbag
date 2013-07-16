export GOBIN=./bin

all: fmt install

install:
	go install ./...

fmt:
	gofmt -w */*.go experiments/cmd/*/*.go
	colcheck */*.go experiments/cmd/*/*.go

push:
	git push origin master
	git push tufts master
	git push github master


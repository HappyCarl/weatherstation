NAME=weatherstation

all: build

build: deps
	go build -o "build/$(NAME)"

run:
	go run weatherstation.go

clean:
	rm -rf build/

deps:

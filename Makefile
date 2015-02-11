NAME=weatherstation
OS=darwin windows linux freebsd
ARCHS=amd64 arm 386

all: build

build: deps
	go build -o "build/$(NAME)"

run:
	go run weatherstation.go

clean:
	rm -rf build/

deps:
	@echo "no deps yet..."

cross: deps setup_cross
	@for GOARCH in $(ARCHS);\
		do \
			for GOOS in $(OS);\
			do\
			GO_ENABLED=0 GOOS=$$GOOS GOARCH=$$GOARCH go build -o "build/$$GOOS/$(NAME).$$GOOS.$$GOARCH";\
			done \
		done

setup_cross:
	@for GOARCH in $(ARCHS);\
		do \
			for GOOS in $(OS);\
			do\
				gvm cross $$GOOS $$GOARCH; \
			done \
		done

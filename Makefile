NAME=weatherstation
OS=darwin windows linux freebsd
ARCHS=amd64 arm 386
TARGET_DIR=build

all: build

build: deps
	go build -o "build/$(NAME)"

run: deps
	go run weatherstation.go

clean:
	rm -rf build/

deps:
	go get github.com/tarm/goserial
	go get code.google.com/p/gcfg

cross: setup_cross deps 
	@for GOARCH in $(ARCHS);\
		do \
			for GOOS in $(OS);\
			do\
			GO_ENABLED=1 GOOS=$$GOOS GOARCH=$$GOARCH go build -o "$(TARGET_DIR)/$$GOOS/$(NAME).$$GOOS.$$GOARCH";\
			done \
		done
	@for ARCH in $(ARCHS);\
		do \
		mv "$(TARGET_DIR)/windows/$(NAME).windows.$$ARCH" "$(TARGET_DIR)/windows/$(NAME).windows.$$ARCH.exe";\
		done

setup_cross:
	@for GOARCH in $(ARCHS);\
		do \
			for GOOS in $(OS);\
			do\
				gvm cross $$GOOS $$GOARCH; \
			done \
		done

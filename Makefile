NAME=weatherstation
OS=darwin windows linux
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
	go get github.com/go-sql-driver/mysql
	go get github.com/jinzhu/gorm
	go get github.com/jteeuwen/go-bindata/...
	go get github.com/elazarl/go-bindata-assetfs/...
	go-bindata-assetfs client/

cross: setup_cross deps
	CC=x86_64-w64-mingw32-gcc CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build -o "$(TARGET_DIR)/windows/$(NAME).windows.64.exe"
	CC=i686-w64-mingw32-gcc CGO_ENABLED=1 GOOS=windows GOARCH=386 go build -o "$(TARGET_DIR)/windows/$(NAME).windows.32.exe"
	CC=arm-linux-gnueabihf-gcc CGO_ENABLED=1 GOOS=linux GOARCH=arm GOARM=6 go build -o "$(TARGET_DIR)/linux/$(NAME).linux.arm" -ldflags="-extld=arm-linux-gnueabihf-gcc"
	#CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -o "$(TARGET_DIR)/$$GOOS/$(NAME).$$GOOS.$$GOARCH"
	#CGO_ENABLED=1 GOOS=darwin GOARCH=386 go build -o "$(TARGET_DIR)/$$GOOS/$(NAME).$$GOOS.$$GOARCH"
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o "$(TARGET_DIR)/linux/$(NAME).linux.amd64"
	#CGO_ENABLED=1 GOOS=linux GOARCH=386 go build -o "$(TARGET_DIR)/$$GOOS/$(NAME).$$GOOS.$$GOARCH"

setup_cross:
	@for GOARCH in $(ARCHS);\
		do \
			for GOOS in $(OS);\
			do\
				gvm cross $$GOOS $$GOARCH; \
			done \
		done

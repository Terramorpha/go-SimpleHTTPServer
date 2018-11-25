commit := $(shell git rev-parse HEAD)
outfile := bin/web
BUILD := go build -ldflags '-X main.gitCommit=$(commit)'


all: build_prod
build_prod: load_files build clean_files

build: main.go
	go build -ldflags '-X main.gitCommit=$(commit)' -o bin/web

install: build
	sudo cp -f bin/web /bin/web


test: build_test runtest

build_test:
	go build -ldflags '-X main.Testing=true -X main.gitCommit=$(commit)' -o bin/web

testr: build runtest_remote


runtest:
	./bin/web -D -V 4 -dir /home/terramorpha

runtest_remote:
	sudo ./bin/web -D -V 4 -dir /home/terramorpha -port 80

runtest_auth: build
	./bin/web -D -V 4 -auth -dir /home/terramorpha

runtest_web: build
	sudo ./bin/web -mode web -D -V 4 -dir ./site -port 80


runtest_fs: build
	./bin/web -mode fileserver -D -V 10 -port 8080 -dir ./fs


install_android:
	GOOS=linux GOARCH=arm64 $(BUILD) -o web_android
	adb push --sync web_android /sdcard/dev/web
	adb shell "su -c 'mount -o remount,rw /system;cp -f /sdcard/dev/web /system/xbin/web;chmod +x /system/xbin/web;mount -o remount,ro /system;exit;'"
	rm web_android


load_files:
	echo "package main" > 1.go
	echo "var HtmlFrontend = \`" >> 1.go
	cat webui/frontend.html >> 1.go
	echo \` >> 1.go

	echo "package main" > 2.go
	echo "var JsFrontend = \`" >> 2.go
	cat webui/frontend.js >> 2.go
	echo \` >> 2.go

clean_files:
	rm -f 1.go 2.go
	echo "package main" > 1.go
	echo "var JsFrontend []byte" >> 1.go
	echo "package main" > 2.go
	echo "var HtmlFrontend []byte" >> 2.go

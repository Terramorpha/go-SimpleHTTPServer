all: install

build: main.go
	go build -o bin/web

install: build
	sudo cp -f web /bin/web


test: build runtest


runtest:
	./web -D -V 4 -dir /home/terramorpha


install_android:
	GOOS=linux GOARCH=arm64 go build -o web_android
	adb push --sync web_android /sdcard/dev/web
	adb shell "su -c 'mount -o remount,rw /system;cp -f /sdcard/dev/web /system/xbin/web;chmod +x /system/xbin/web;mount -o remount,ro /system;exit;'"



build_win_amd64:
	GOOS=windows GOARCH=amd64 go build -o bin/win_amd64.exe


build_linux_amd64:
	GOOS=linux GOARCH=amd64 go build -o bin/linux_amd64

build_linux_arm64
	GOOS=linux GOARCH=arm64 go build -o bin/linux_arm64




build_all: build_linux_amd64 build_win_amd64

test_auth: build
	./web -auth -p pass -u user -D -V 4

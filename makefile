all: install

build: main.go
	go build -o bin/web

install: build
	sudo cp -f bin/web /bin/web


test: build runtest

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
	GOOS=linux GOARCH=arm64 go build -o web_android
	adb push --sync web_android /sdcard/dev/web
	adb shell "su -c 'mount -o remount,rw /system;cp -f /sdcard/dev/web /system/xbin/web;chmod +x /system/xbin/web;mount -o remount,ro /system;exit;'"



build_win_amd64:
	GOOS=windows GOARCH=amd64 go build -o bin/win_amd64.exe


build_linux_amd64:
	GOOS=linux GOARCH=amd64 go build -o bin/linux_amd64

build_linux_arm64:
	GOOS=linux GOARCH=arm64 go build -o bin/linux_arm64




build_all: build_linux_amd64 build_win_amd64



sync:
	git add .
	git commit -m "sync automated"
	git push
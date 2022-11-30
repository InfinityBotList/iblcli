all:
	go build -v
publish:
	mkdir -p bin bin/linux bin/darwin bin/windows
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/linux/amd64/ibl
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o bin/darwin/amd64/ibl
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o bin/darwin/arm64/ibl
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o bin/windows/amd64/ibl.exe
	mkdir -p /iblseeds/shadowsight
	mv -vf bin/* /iblseeds/shadowsight
	rm -rf bin
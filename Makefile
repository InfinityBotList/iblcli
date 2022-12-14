# Build revision should be unique for each build AND not the git commit hash
# because we want to be able to build the same commit multiple times and
# have the build revision be different each time.

BUILDREV := $(shell uuidgen)
BUILDTIME := $(shell date '+%Y-%m-%d %H:%M:%S')
REPONAME := github.com/InfinityBotList/ibl
GOFLAGS := -ldflags="-X '$(REPONAME)/cmd.BuildRev=$(BUILDREV)' -X '$(REPONAME)/cmd.BuildTime=$(BUILDTIME)'"

COMBOS := linux/386 linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64 windows/386

all:
	go build -v $(GOFLAGS)
publish:
	mkdir bin

	for combo in $(COMBOS); do \
		echo "$$combo"; \
		CGO_ENABLED=0 GOOS=$${combo%/*} GOARCH=$${combo#*/} go build -o bin/$$combo/ibl $(GOFLAGS); \
	done

	# Rename all the windows binaries to .exe
	for folder in bin/windows/*; do \
		mv -vf $$folder/ibl $$folder/ibl.exe; \
	done

	rm -rf /iblseeds/shadowsight
	mkdir -p /iblseeds/shadowsight
	mv -vf bin/* /iblseeds/shadowsight
	echo $(BUILDREV) > /iblseeds/shadowsight/current_rev
	rm -rf bin
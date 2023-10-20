# Build revision should be unique for each build AND not the git commit hash
# because we want to be able to build the same commit multiple times and
# have the build revision be different each time.

BUILDTIME := $(shell date '+%Y-%m-%d | %H:%M:%S')
BUILDREV := ${BUILDTIME}
REPONAME := github.com/InfinityBotList/ibl
PROJECTNAME := iblcli
GOFLAGS := -trimpath -ldflags="-s -w -X '$(REPONAME)/cmd.BuildRev=$(BUILDREV)' -X '$(REPONAME)/cmd.BuildTime=$(BUILDTIME)' -X '$(REPONAME)/cmd.ProjectName=$(PROJECTNAME)'"
GOFLAGS_DBG := -trimpath -ldflags="-X '$(REPONAME)/cmd.BuildRev=$(BUILDREV)' -X '$(REPONAME)/cmd.BuildTime=$(BUILDTIME)'"
CDN_PATH := /silverpelt/cdn/ibl/dev

COMBOS := linux/386 linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64 windows/386 freebsd/amd64

all:
	CGO_ENABLED=0 go build -v $(GOFLAGS_DBG)
install: all
	cp -rf ibl /usr/bin/ibl
publish:
	mkdir -p bin

	for combo in $(COMBOS); do \
		echo "$$combo"; \
		CGO_ENABLED=0 GOOS=$${combo%/*} GOARCH=$${combo#*/} go build -o bin/$$combo/ibl $(GOFLAGS); \
		sha512sum bin/$$combo/ibl > bin/$$combo/ibl.sha512; \
	done

	# Rename all the windows binaries to .exe
	for folder in bin/windows/*; do \
		mv -vf $$folder/ibl $$folder/ibl.exe; \
	done

	rm -rf $(CDN_PATH)/downloads/iblcli
	mkdir -p $(CDN_PATH)/downloads/iblcli
	mv -vf bin/* $(CDN_PATH)/downloads/iblcli
	echo -n "$(BUILDREV)" > $(CDN_PATH)/downloads/iblcli/current_rev
	echo -n "Infinity Bot List Developer Client.\nThis is a developer only client for managing our infrastructure (databases etc.). You probably want IBLCLI instead" > /silverpelt/cdn/ibl/dev/downloads/iblcli/description
	chown -Rv ibl:ibl $(CDN_PATH)
	rm -rf bin

mkiblseed:
	su ibl -c "cd ~/ && ibl db new seed $(CDN_PATH)/seed.iblseed --db infinity --backup-tables changelogs,partner_types"
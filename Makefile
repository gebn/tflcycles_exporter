GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
GOARM := $(shell go env GOARM)

OUT := tflcycles_exporter
VERSION := $(shell ./bin/version)

# Ignore any C/C++ toolchain present.
export CGO_ENABLED = 0

arm := $(GOARM)
ifneq ($(arm),)
    arm := v$(arm)
endif
ARCHIVE_BASE := $(OUT)-$(VERSION:v%=%).$(GOOS)-$(GOARCH)$(arm)

ifeq ($(GOOS), windows)
	OUT := $(OUT).exe
	ARCHIVE := $(ARCHIVE_BASE).zip
else
	ARCHIVE := $(ARCHIVE_BASE).tar.gz
endif

LDFLAGS := -ldflags=" \
-X 'github.com/gebn/go-stamp/v2.User=$(shell whoami)' \
-X 'github.com/gebn/go-stamp/v2.Host=$(shell hostname)' \
-X 'github.com/gebn/go-stamp/v2.timestamp=$(shell date +%s)' \
-X 'github.com/gebn/go-stamp/v2.Commit=$(shell git rev-parse HEAD)' \
-X 'github.com/gebn/go-stamp/v2.Branch=$(shell git rev-parse --abbrev-ref HEAD)' \
-X 'github.com/gebn/go-stamp/v2.Version=$(VERSION)'"

build:
	go build $(LDFLAGS) -o $(OUT) ./cmd/tflcycles_exporter

dist: build
	mkdir $(ARCHIVE_BASE)
	mv $(OUT) $(ARCHIVE_BASE)/
	cp LICENSE $(ARCHIVE_BASE)/
ifeq ($(GOOS), windows)
	zip -r $(ARCHIVE) $(ARCHIVE_BASE)
else
	tar -czf $(ARCHIVE) $(ARCHIVE_BASE)
endif
	rm -r $(ARCHIVE_BASE)

test:
	go test ./...

# Used by CI to get the path of the archive created by `make dist`.
distpath:
	@echo $(ARCHIVE)

# Used by CI to get the name of the tag to push to the image registry.
version:
	@echo $(VERSION)

clean:
	rm -f $(OUT) $(ARCHIVE_BASE) $(ARCHIVE)

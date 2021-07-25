.EXPORT_ALL_VARIABLES:
APP=mov-pkg
APP_COMMIT:=$(shell git rev-parse HEAD)
APP_EXECUTABLE="./out/$(APP)"
TARGET:=mov-pkg
PKG ?=-v /vendor
ALL_PACKAGES=$(shell go list ./... | grep $(PKG))
GO111MODULE:=on
SOURCE_DIRS=$(shell go list ./... | grep -v /vendor | grep -v /out | cut -d "/" -f4 | uniq)
GOPATH_BIN:=$(shell go env GOPATH)/bin

all: clean	build

list-pkg:
	: $(ALL_PACKAGES)

clean:
	rm -rf out/

gen-version:
	git describe --tags --abbrev=0 | sed 's/^v//' > .version

build: gen-version
	$(eval VERSION:=$(shell cat .version))
	mkdir -p out/
	go build -o out/$(TARGET) -ldflags "-X 'main.version=$(VERSION)' -X 'main.commit=$(COMMIT)'"

test:
	go test -test.short -count=1 -cover $(ALL_PACKAGES)

fmtcheck:
	@gofmt -l -s $(SOURCE_DIRS) | grep ".*\.go"; if [ "$$?" = "0" ]; then exit 1; fi

vet:
	@go vet ./...

fixfmt:
	@gofmt -w .
	@goimports -w .

try:
	go run mov-pkg.go -f github.com/firodj/mov-pkg/examples -t github.com/firodj/mov-pkg/examples/models -s Temp github.com/firodj/mov-pkg/examples

drytry:
	go run mov-pkg.go -d -f github.com/firodj/mov-pkg/examples -t github.com/firodj/mov-pkg/examples/models -s Temp github.com/firodj/mov-pkg/examples

.PHONY:	all

TARGET    = bin/dnscheck
DEPS      = $(shell find . -type f -name '*.go')

MANUL_BIN = $(GOPATH)/bin/manul

COMMIT_ID   = $(shell git describe --tags --always --dirty=-dev)
COMMIT_UNIX = $(shell git show -s --format=%ct HEAD)
BUILD_COUNT = $(shell git rev-list --count HEAD)
BUILD_UNIX  = $(shell date +%s)

.PHONY: default
default: $(TARGET)

$(TARGET): $(DEPS)
	mkdir -p $(dir $@)
	cd cmd/dnscheck && go build -o ../../$@

.PHONY: deps
deps: $(MANUL_BIN)
	$(MANUL_BIN) -U github.com/go-sql-driver/mysql
	$(MANUL_BIN) -U github.com/miekg/dns
	$(MANUL_BIN) -U github.com/oschwald/geoip2-golang
	$(MANUL_BIN) -U gopkg.in/yaml.v2
	$(MANUL_BIN) -Q

$(MANUL_BIN):
	go get -u github.com/kovetskiy/manul

.PHONY: test
test:
	go test $(shell go list './...' | grep -v '/vendor/')

.PHONY: clean
clean:
	rm -f $(TARGET)

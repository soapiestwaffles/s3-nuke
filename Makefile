GOCMD=go
GOTEST=$(GOCMD) test
GOVET=$(GOCMD) vet
EXPORT_RESULT?=false # for CI please set EXPORT_RESULT to true

GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
WHITE  := $(shell tput -Txterm setaf 7)
CYAN   := $(shell tput -Txterm setaf 6)
RESET  := $(shell tput -Txterm sgr0)

.PHONY: all test

all: help

## Build:
build-all: s3-nuke s3-gen s3-metrics ## Build all binaries
	@echo "All binaries built..."

s3-nuke: ## Build s3-nuke binary
	@echo "Building s3-nuke..."
	@$(GOCMD) build -o bin/s3-nuke ./main.go

s3-gen: ## Build s3-gen binary
	@echo "Building s3-gen..."
	@$(GOCMD) build -o bin/s3-gen ./tools/s3-gen/main.go

s3-metrics: ## Build s3-metrics binary
	@echo "Building s3-metrics..."
	@$(GOCMD) build -o bin/s3-metrics ./tools/s3-metrics/main.go

clean:  ## Clean up after builds
	@rm bin/s3-nuke || true
	@rm bin/s3-gen || true
	@rm bin/s3-metrics || true
	@rmdir bin

## Run:
run-s3-gen: ## Run s3-gen from source
	@$(GOCMD) run ./tools/s3-gen/main.go

run-s3-metrics: ## Run s3-metrics from source
	@$(GOCMD) run ./tools/s3-metrics/main.go

## Test:
test: ## Run tests
ifeq ($(EXPORT_RESULT), true)
	GO111MODULE=off go get -u github.com/jstemmer/go-junit-report
	$(eval OUTPUT_OPTIONS = | tee /dev/tty | go-junit-report -set-exit-code > junit-report.xml)
endif
	$(GOTEST) -v -race ./... $(OUTPUT_OPTIONS)

coverage: ## Run tests and export coverage
	@$(GOTEST) -cover -covermode=count -coverprofile=coverage.out ./...
	@$(GOCMD) tool cover -func coverage.out 
ifeq ($(EXPORT_RESULT), true)
	# Output to HTML
	@$(GOCMD) tool cover -html=coverage.out -o coverage.html
	
	# Output to XML
	@GO111MODULE=off go get -u github.com/AlekSi/gocov-xml
	@GO111MODULE=off go get -u github.com/axw/gocov/gocov
	@gocov convert coverage.out | gocov-xml > coverage.xml
endif

## Lint:
lint-go: ## run golangci-lint on project
	@$(eval OUTPUT_OPTIONS = $(shell [ "${EXPORT_RESULT}" == "true" ] && echo "--out-format checkstyle ./... | tee /dev/tty > checkstyle-report.xml" || echo "" ))
	docker run --rm -v $(shell pwd):/app -w /app golangci/golangci-lint:latest-alpine golangci-lint run --deadline=500s $(OUTPUT_OPTIONS)

# lint-yaml: ## lint yaml files in project
# ifeq ($(EXPORT_RESULT), true)
# 	GO111MODULE=off go get -u github.com/thomaspoignant/yamllint-checkstyle
# 	$(eval OUTPUT_OPTIONS = | tee /dev/tty | yamllint-checkstyle > yamllint-checkstyle.xml)
# endif
# 	docker run --rm -it -v $(shell pwd):/data cytopia/yamllint -f parsable $(shell git ls-files '*.yml' '*.yaml') $(OUTPUT_OPTIONS)

lint-dockerfile: ## lint Dockerfile
ifeq ($(shell test -e ./build/Dockerfile && echo -n yes),yes)
	$(eval CONFIG_OPTION = $(shell [ -e $(shell pwd)/.hadolint.yaml ] && echo "-v $(shell pwd)/.hadolint.yaml:/root/.config/hadolint.yaml" || echo "" ))
	$(eval OUTPUT_OPTIONS = $(shell [ "${EXPORT_RESULT}" == "true" ] && echo "--format checkstyle" || echo "" ))
	$(eval OUTPUT_FILE = $(shell [ "${EXPORT_RESULT}" == "true" ] && echo "| tee /dev/tty > checkstyle-report.xml" || echo "" ))
	docker run --rm -i $(CONFIG_OPTION) hadolint/hadolint hadolint $(OUTPUT_OPTIONS) - < ./build/Dockerfile $(OUTPUT_FILE)
endif

## Clean:
test-clean: ## Clean up after tests
	@echo cleaning up...
	@rm coverage.html coverage.out coverage.xml || true

## Help:
help: ## Show this help.
	@echo ''
	@echo 'Usage:'
	@echo '  ${YELLOW}make${RESET} ${GREEN}<target>${RESET}'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} { \
		if (/^[a-zA-Z_-]+:.*?##.*$$/) {printf "    ${YELLOW}%-20s${GREEN}%s${RESET}\n", $$1, $$2} \
		else if (/^## .*$$/) {printf "  ${CYAN}%s${RESET}\n", substr($$1,4)} \
		}' $(MAKEFILE_LIST)

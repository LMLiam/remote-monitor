.DEFAULT_GOAL := check

SHFMT_VERSION ?= v3.13.1
SHELLCHECK_IMAGE ?= docker.io/koalaman/shellcheck:v0.11.0
SHELLCHECK_DOCKER = docker run --rm -v "$$PWD:/mnt" -w /mnt $(SHELLCHECK_IMAGE)

.PHONY: check fmt shfmt shellcheck scripts vet test lint build generate setup help

check: fmt shfmt shellcheck scripts vet test lint build

fmt:
	@unformatted="$$(gofmt -l ./cmd ./internal ./tests)"; \
	if [ -n "$$unformatted" ]; then \
		echo "Go files need gofmt:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

shfmt:
	go run "mvdan.cc/sh/v3/cmd/shfmt@$(SHFMT_VERSION)" -i 2 -ci -d .github/scripts internal/transport/sampler internal/transport/sampler.sh tests/e2e/ssh-target

shellcheck:
	@command -v docker >/dev/null || { echo "docker is required for make shellcheck"; exit 1; }
	$(SHELLCHECK_DOCKER) --version
	$(SHELLCHECK_DOCKER) \
		-S warning -s bash \
		.github/scripts/*.sh \
		tests/e2e/ssh-target/*.sh \
		internal/transport/sampler/assemble.sh \
		internal/transport/sampler.sh
	@awk 'NF && $$1 !~ /^#/ { print "internal/transport/sampler/" $$0 }' internal/transport/sampler/manifest.txt | \
		xargs $(SHELLCHECK_DOCKER) -S warning -s bash -e SC2034,SC2154

scripts:
	bash .github/scripts/test-conventional-title.sh
	bash .github/scripts/test-next-release-tag.sh
	bash .github/scripts/test-verify-main-checks.sh
	bash .github/scripts/test-build-workflow.sh
	bash .github/scripts/test-publish-wiki.sh

vet:
	go vet -tags=integration ./...

test:
	go test -tags=integration ./...

lint:
	golangci-lint run --build-tags=integration

build:
	go build -o /tmp/remote-monitor ./cmd/remote-monitor

generate:
	go generate ./internal/transport

setup:
	bash .github/scripts/install-git-hooks.sh

help:
	@printf '%s\n' \
		'make check       Run the full local check gate' \
		'make fmt         Check Go formatting' \
		'make shfmt       Check shell formatting with shfmt' \
		'make shellcheck  Run ShellCheck via Docker' \
		'make scripts     Test workflow helper scripts' \
		'make vet         Run go vet with integration tags' \
		'make test        Run Go tests with integration tags' \
		'make lint        Run golangci-lint with integration tags' \
		'make build       Build /tmp/remote-monitor' \
		'make generate    Regenerate generated sampler artifacts' \
		'make setup       Install local Git hooks'

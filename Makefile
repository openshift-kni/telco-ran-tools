RUNTIME ?= podman
REPOOWNER ?= openshift-kni
IMAGENAME ?= telco-ran-tools
IMAGETAG ?= latest

all: dist

.PHONY: build
build: dist

.PHONY: ci-job-e2e
ci-job-e2e: test-e2e

.PHONY: ci-job-unit
ci-job-unit: test-unit

outdir:
	mkdir -p _output || :

.PHONY: deps-update
deps-update:
	go mod tidy && go mod vendor

.PHONY: deps-clean
deps-clean:
	rm -rf vendor

.PHONY: dist
dist: binaries

.PHONY: binaries
binaries: outdir deps-update
	# go flags are set in here
	./hack/build-binaries.sh

.PHONY: clean
clean:
	rm -rf _output

.PHONY: image
image:
	@echo "building image"
	$(RUNTIME) build -f Dockerfile -t quay.io/$(REPOOWNER)/$(IMAGENAME):$(IMAGETAG) .

.PHONY: push
push: image
	@echo "pushing image"
	$(RUNTIME) push quay.io/$(REPOOWNER)/$(IMAGENAME):$(IMAGETAG)

.PHONY: test-unit
test-unit: test-unit-cmd

.PHONY: test-unit-cmd
test-unit-cmd:
	go test ./cmd/...

.PHONY: test-e2e
test-e2e: binaries
	ginkgo test/e2e

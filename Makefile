SHELL ?= /bin/bash
RUNTIME ?= podman
DOCKERFILE ?= Dockerfile
REPOOWNER ?= openshift-kni
IMAGENAME ?= telco-ran-tools
IMAGETAG ?= latest

all: dist

.PHONY: fmt
fmt: ## Run go fmt against code.
	@echo "Running go fmt"
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	@echo "Running go vet"
	go vet ./...

.PHONY: shellcheck
shellcheck: ## Run shellcheck
	@echo "Running shellcheck"
	hack/shellcheck.sh

.PHONY: bashate
bashate: ## Run bashate
	@echo "Running bashate"
	hack/bashate.sh

.PHONY: update-resources
update-resources: shellcheck bashate
	@echo "Updating docs/resources/boot-beauty.ign"
	@sed -i "/\"path\":.*extract-ocp/,/\"path\":/ s#base64,.*#base64,$(shell base64 -w 0 docs/resources/extract-images.sh)\"#" docs/resources/boot-beauty.ign
	@echo "Updating docs/resources/discovery-beauty.ign"
	@sed -i "/\"path\":.*extract-ai/,/\"path\":/ s#base64,.*#base64,$(shell base64 -w 0 docs/resources/extract-images.sh)\"#" docs/resources/discovery-beauty.ign
	@hack/update-docs.sh

.PHONY: check-git-tree
check-git-tree: # If generated code is added in the future, add generation dependency here
	hack/check-git-tree.sh

.PHONY: golangci-lint
golangci-lint: ## Run golangci-lint against code.
	@echo "Running golangci-lint"
	hack/golangci-lint.sh

.PHONY: build
build: dist

.PHONY: ci-job-e2e
ci-job-e2e: test-e2e check-git-tree

.PHONY: ci-job-unit
ci-job-unit: fmt vet test-unit golangci-lint shellcheck bashate update-resources check-git-tree

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
binaries: outdir deps-update fmt vet
	# go flags are set in here
	./hack/build-binaries.sh

.PHONY: clean
clean:
	rm -rf _output

.PHONY: image
image:
	@echo "building image"
	$(RUNTIME) build -f $(DOCKERFILE) -t quay.io/$(REPOOWNER)/$(IMAGENAME):$(IMAGETAG) .

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

PULL_BASE_SHA ?= origin/main # Allow the template check base ref to be overridden

.PHONY: markdownlint-image
markdownlint-image:  ## Build local container markdownlint-image
	$(RUNTIME) image build -f ./hack/Dockerfile.markdownlint --tag $(IMAGENAME)-markdownlint:latest

.PHONY: markdownlint-image-clean
markdownlint-image-clean:  ## Remove locally cached markdownlint-image
	$(RUNTIME) image rm $(IMAGENAME)-markdownlint:latest

markdownlint: markdownlint-image  ## run the markdown linter
	$(RUNTIME) run \
		--rm=true \
		--env RUN_LOCAL=true \
		--env VALIDATE_MARKDOWN=true \
		--env PULL_BASE_SHA=$(PULL_BASE_SHA) \
		-v $$(pwd):/workdir:Z \
		$(IMAGENAME)-markdownlint:latest


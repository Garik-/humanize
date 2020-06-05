include ../../includes.mk

APP := humanize

.PHONY: build package lint test clean

build:
	@echo "=> building $(APP) binary"
	@$(GO_FLAGS) $(GO_LDFLAGS) $(GO) build -a -o $(BIN_DIR)/$(APP) .


package:
	@echo "=> packaging $(DOCKER_REPO)/$(APP):$(VERSION)"
	@docker build -t $(DOCKER_REPO)/$(APP):$(VERSION) -f $(DOCKERFILE) $(DOCKER_CONTEXT) $(LOG_OUTPUT)


lint:   bootstrap ## run golangci-linter
	@echo "=> linting codebase"
	@golangci-lint run ./...


test:   lint ## run all test suites
	@echo "=> running tests"
	@cd ../../pkg/midi; go test -race -coverprofile=../../coverage.txt -covermode=atomic ./...


clean:
	@rm -f humanize coverage.txt


deploy:
	@echo "=> deploy $(APP)"


promote: registry-login ## promote artefact
	@echo "=> release"
	@docker pull $(DOCKER_REPO)/$(APP):master-$(GIT_TAG_HASH)
	@docker tag $(DOCKER_REPO)/$(APP):master-$(GIT_TAG_HASH) $(DOCKER_REPO)/$(APP):$(VERSION)
	@docker push $(DOCKER_REPO)/$(APP):$(VERSION)


publish: registry-login ## publish docker image
	@echo "=> pushing $(DOCKER_IMAGE)"
	@docker push $(DOCKER_IMAGE)
ifeq (${DOCKER_TAG_LATEST},true)
	@docker tag $(DOCKER_IMAGE) $(DOCKER_REPO)/$(APP):latest
	@docker push $(DOCKER_REPO)/$(APP):latest
endif
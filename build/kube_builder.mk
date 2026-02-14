#
# build/kube_builder.mk - Docker build/push for operator/manager image
#
# Include this in projects that build a Kubernetes operator/manager image
# (e.g. kubebuilder, operator-sdk). Uses IMG and LATESTIMG from vars.mk.
#
# Set in your Makefile before including:
#   IMG ?= user/repo:$(VERSION)
#   LATESTIMG ?= user/repo:latest
# Optional: make docker-build registry=myreg.io
#
.PHONY: docker-build docker-push
docker-build: test ## Build docker image with the manager.
ifneq ($(registry),)
	$(DOCKER) build -t $(registry)/$(IMG) -t $(registry)/$(LATESTIMG) .
else
	$(DOCKER) build -t $(IMG) -t $(LATESTIMG) .
endif

docker-push: ## Push docker image with the manager.
ifneq ($(registry),)
	$(DOCKER) push $(registry)/$(IMG)
else
	$(DOCKER) push $(IMG)
endif

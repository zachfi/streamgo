#
# Makefile fragment for Docker actions (login, generic build, push, snapshot)
#
# For operator/manager image build and push, use build/kube_builder.mk instead.
# Expects build/vars.mk for IMG, LATESTIMG, DOCKER.
#
# Set registry (e.g. make docker-build registry=localhost:5000) to build and
# push with a prefix for local development and testing.
#

# When registry is set, tag and push as registry/IMG and registry/LATESTIMG
DOCKER_IMG       = $(if $(registry),$(registry)/$(IMG),$(IMG))
DOCKER_LATESTIMG = $(if $(registry),$(registry)/$(LATESTIMG),$(LATESTIMG))

docker-login:
	@echo "=== $(PROJECT_NAME) === [ docker-login     ]: logging into docker hub"
	@if [ -z "$${DOCKER_USERNAME}" ]; then \
		echo "Failure: DOCKER_USERNAME not set" ; \
		exit 1 ; \
	fi
	@if [ -z "$${DOCKER_PASSWORD}" ]; then \
		echo "Failure: DOCKER_PASSWORD not set" ; \
		exit 1 ; \
	fi
	@echo "=== $(PROJECT_NAME) === [ docker-login     ]: username: '$$DOCKER_USERNAME'"
	@echo $${DOCKER_PASSWORD} | $(DOCKER) login -u $${DOCKER_USERNAME} --password-stdin

docker:
	$(DOCKER) build -t $(DOCKER_IMG) -t $(DOCKER_LATESTIMG) .

docker-push:
	$(DOCKER) push $(DOCKER_IMG)

docker-snapshot: docker docker-push

.PHONY: docker-login docker docker-push docker-snapshot

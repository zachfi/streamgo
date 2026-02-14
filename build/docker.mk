#
# Makefile fragment for Docker actions (login, generic build, snapshot)
#
# For operator/manager image build and push, use build/kube_builder.mk instead.
# Expects build/vars.mk for IMG, LATESTIMG, PROJECT_VER, DOCKER.
#

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
	$(DOCKER) build -t $(IMG) .

docker-snapshot: docker
	$(DOCKER) tag $(IMG) $(LATESTIMG)
	$(DOCKER) push $(IMG)

.PHONY: docker-login docker docker-snapshot

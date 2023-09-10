#
# Makefile fragment for Docker actions
#
DOCKER            ?= docker

docker-login:
	@echo "=== $(PROJECT_NAME) === [ docker-login     ]: logging into docker hub"
	@if [ -z "${DOCKER_USERNAME}" ]; then \
		echo "Failure: DOCKER_USERNAME not set" ; \
		exit 1 ; \
	fi
	@if [ -z "${DOCKER_PASSWORD}" ]; then \
		echo "Failure: DOCKER_PASSWORD not set" ; \
		exit 1 ; \
	fi
	@echo "=== $(PROJECT_NAME) === [ docker-login     ]: username: '$$DOCKER_USERNAME'"
	@echo ${DOCKER_PASSWORD} | $(DOCKER) login -u ${DOCKER_USERNAME} --password-stdin

docker:
	docker build -t zachfi/streamgo .

docker-snapshot: docker
	docker tag zachfi/streamgo:latest zachfi/streamgo:${PROJECT_VER}
	docker push zachfi/streamgo:${PROJECT_VER}


.PHONY: docker-login

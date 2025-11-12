## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'


## audit: check code is ok
.PHONY: audit
audit:
	go vet ./...
	go tool staticcheck ./...
	go tool govulncheck


## build: build & push image
.PHONY: build
build: audit
	fly deploy -a $(FLY_APP_NAME)  -i $$(KO_DOCKER_REPO=registry.fly.io/$(FLY_APP_NAME) go tool ko build . --bare --platform=linux/amd64)
#	KO_DOCKER_REPO=registry.fly.io/$(FLY_APP_NAME) fly deploy -a $(FLY_APP_NAME)  -i $$(go tool ko build . --bare --platform=linux/amd64)

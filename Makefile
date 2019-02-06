VERSION := $(shell git describe --tags)

.PHONY: prepare_test
prepare_test:
	# docker cp conf/test_daemon_settings.yml $(shell docker-compose ps -q test_daemon):/storage/data/daemon_settings.yml
	docker-compose up --no-start test_lbrynet
	docker cp conf/daemon_settings.yml $(shell docker-compose ps -q test_lbrynet):/storage/data/daemon_settings.yml
	docker-compose start test_daemon

.PHONY: test
test:
	go test ./...

.PHONY: test_circleci
test_circleci:
	scripts/wait_for_wallet.sh
	go test ./...

release:
	goreleaser --rm-dist

snapshot:
	goreleaser --snapshot --rm-dist

.PHONY: image
image:
	docker build -t lbryweb/lbryweb-go:$(VERSION) -t lbryweb/lbryweb-go:latest .

.PHONY: publish_image
publish_image:
	docker push lbryweb/lbryweb-go

embed:
	rice embed-go -i ./routes

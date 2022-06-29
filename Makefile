all: update-deps build
build:
	go build -mod vendor -o github-slack-bot .
.PHONY: build

debug:
	go build -gcflags="all=-N -l"  -mod vendor -o github-slack-bot .
.PHONY: build

update-deps:
	GO111MODULE=on go mod vendor
.PHONY: update-deps

run: check-env
	./github-slack-bot
.PHONY: run

clean:
	rm -rf ./github-slack-bot
.PHONY: clean

check-env:
ifndef SLACK_APP_TOKEN
	$(error SLACK_APP_TOKEN is undefined)
endif
ifndef SLACK_BOT_TOKEN
	$(error SLACK_BOT_TOKEN is not set)
endif
ifndef GITHUB_OAUTH_TOKEN
	$(error GITHUB_OAUTH_TOKEN is not set)
endif
ifndef GITHUB_ORG
	$(error GITHUB_ORG is not set)
endif
ifndef GITHUB_REPO
	$(error GITHUB_REPO is not set)
endif


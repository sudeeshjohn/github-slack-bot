FROM golang:1.18.10 AS builder
WORKDIR /go/src/github.com/github-slack-bot
COPY . .
RUN make
RUN pwd
FROM quay.io/powercloud/all-in-one
RUN ls -lah
COPY --from=builder /go/src/github.com/github-slack-bot/github-slack-bot /usr/bin/
ENTRYPOINT ["/usr/bin/github-slack-bot"]

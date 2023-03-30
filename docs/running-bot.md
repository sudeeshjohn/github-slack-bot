# Running bot

You'll need slack app, github auth tokens, organization (or owner) name and also the repository name before starting.
Before proceeding you need to [create the slack app](docs/preparing-app.md).

There are multiple ways to this bot

## Running bot locally

export these environment variable s

```
export SLACK_APP_TOKEN=<>
export SLACK_BOT_TOKEN=<>
export GITHUB_OAUTH_TOKEN=<>
export GITHUB_ORG=<>
export GITHUB_REPO=<>
export GITHUB_ENTERPRISE_URL=<https://github.xyz.com/api/v3/>
```

set GITHUB_ENTERPRISE_URL only if you are planning to interact with an enterprise git

```
make all && make run
```

## Running bot in a container

export below environment variables

```
export SLACK_APP_TOKEN=<>
export SLACK_BOT_TOKEN=<>
export GITHUB_OAUTH_TOKEN=<>
export GITHUB_ORG=<>
export GITHUB_REPO=<>
export GITHUB_ENTERPRISE_URL=<https://github.xyz.com/api/v3/>
```

```
docker run -it -eGITHUB_ENTERPRISE_URL=$GITHUB_ENTERPRISE_URL \
-e SLACK_APP_TOKEN=$SLACK_APP_TOKEN -e SLACK_BOT_TOKEN=$SLACK_BOT_TOKEN \
-e GITHUB_OAUTH_TOKEN=$GITHUB_OAUTH_TOKEN -e GITHUB_ORG=$GITHUB_ORG \
-e GITHUB_REPO=$GITHUB_REPO github-slack-bot:0.1
```

## Running the bot to k8s cluster as a pod

update config/secret.yaml with base64 values. you can convert token and other values to base64 like below

```
BASE64_SLACK_APP_TOKEN=$(echo -n $SLACK_APP_TOKEN | base64)
```

Create k8s secrets

```
kubectl apply -f config/secret.yaml
```

Run the pod

```
kubectl apply -f config/pod.yaml
```

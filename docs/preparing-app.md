# Create App

This app works by using slack's [Socket Mode](https://api.slack.com/apis/connections/socket) connection protocol.

To get started, you must have or create a [Slack App](https://api.slack.com/apps?new_app=1) and enable `Socket Mode`,
which will generate your app token (`SLACK_APP_TOKEN` and `SLCK_BOT_TOKEN`) that will be needed to authenticate.

Additionally, you need to subscribe to events for your bot to respond to under the `Event Subscriptions` section. Common
event subscriptions for bots include `app_mention` or `message.im`.

After setting up your subscriptions, add scopes necessary to your bot in the `OAuth & Permissions`. The following scopes
are recommended for getting started, though you may need to add/remove scopes depending on your bots purpose:

* `app_mentions:read`
* `channels:history`
* `chat:write`
* `groups:history`
* `im:history`
* `mpim:history`

Once you've selected your scopes install your app to the workspace and navigate back to the `OAuth & Permissions`
section. Here you can retrieve yor bot's OAuth token (`SLACK_BOT_TOKEN` in the examples) from the top of the page.


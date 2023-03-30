# github-slack-bot

The **@github-slack-bot** is a Slack App that allows users to interact with GitHub Repositories.

## Setup

* [Settingup Slack App](docs/preparing-app.md)
* [Running the bot](docs/running-bot.md)

To see the available commands, type `help`.

Examples:

1. Get github user details

   member get <user name>
   ```
   Eg: member get sudeeshjohn
   Output:
    Login:    sudeeshjohn
    Name:    SUDEESH JOHN
    Email:    sudeeshjohn@gmail.com
    ID:    38642
    Public Repos:    48
   ```

2. Add a user to a team member add <user name> team=<team name>
    ```
   Eg:
   member add sudeeshjohn team=xyz
   ```
3. List all issues those are assigned
    ```
    issue list assigned
    ```
4. List all issues assigned to a user and also not updated since a date issue list assignedto username=<user>
   ,noupdatesince=<date string>
   date string must be in yyyy-mm-dd (eg: 2022-01-01) format


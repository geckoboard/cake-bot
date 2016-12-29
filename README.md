# cake-bot [![Deploy](https://www.herokucdn.com/deploy/button.svg)](https://heroku.com/deploy)

Cake bot is your friendly neighbourhood code review assistant!

At [Geckoboard](https://www.geckoboard.com) we do code review on just about every change that's deployed.
This process isn't meant for making it easier to micro-manage other people's
work. Instead it's meant to help ensure that at least several members of the
team know how the platform is changing, and provides an opportunity for us to
help each other see problems from a different perspective.

[This article](http://glen.nu/ramblings/oncodereview.php) goes into a
lot of detail about a style of code review that's very close to what we
use at Geckoboard.

Cake bot listens for submitted code review webhook pings from Github and
pings the author of the PR in slack. Cake bot can use information in the
slack team directory to work out which slack users match up with which
Github users. Create a profile field in your slack team called `github`
and ask users to enter their Github username in it.

## Configuration

At a minimum you will need to specify these environment variables:

- `GITHUB_ACCESS_TOKEN` A personal access token of a user that has write
  access to your repos. We recommend creating a separate "bot user" for
  this.
- `SLACK_TOKEN` An API key that can scan your slack's team directory.
- `SLACK_HOOK` The URL of an incoming slack hook that can post messages
  to a room in your slack team.

During development you can set these variables in a `.env` file in the
current working directory. Cake bot will set these as environment
variables.

## Testing

```console
$ make test
```

You can also send some example webhooks to the server during development:

```console
$ cat example-webhooks/pull_request_review_approved.json | curl http://localhost:8090 \
  -X POST \
  -H "X-Github-Event: pull_request_review" \
  -d @-
```

## Running

```console
$ make run
```

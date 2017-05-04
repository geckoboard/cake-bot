# cake-bot [![Deploy](https://www.herokucdn.com/deploy/button.svg)](https://heroku.com/deploy)

Cake bot is your friendly neighbourhood code review assistant!

At [Geckoboard](https://www.geckoboard.com) we do code review on just
about every change that's deployed. This process isn't meant for making
it easier to micro-manage other people's work. Instead it's meant to
help ensure that at least several members of the team know how the
platform is changing, and provides an opportunity for us to help each
other see problems from a different perspective.

[This article](http://glen.nu/ramblings/oncodereview.php) goes into a
lot of detail about a style of code review that's very close to what we
use at Geckoboard.

Cake Bot listens for GitHub events and notifies your team on Slack. The
following notifications are currently supported:

- When a review is requested, Cake Bot will notify the person that has
  been chosen for the review;
- When someone reviews a pull request, Cake Bot will notify the pull
  request author about any new comments;
- When a pull request is accepted by the reviewer, the PR author will be
  notified that they have cake! (Figuratively)

Cake bot can use information in the slack team directory to work out
which slack users match up with which GitHub users. Create a profile
field in your Slack team called `github` and ask users to enter their
GitHub username in it.

## Configuration

At a minimum you will need to specify these environment variables:

- `PORT` The HTTP port on which to listen for webhooks.
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
$ cat example-webhooks/pull_request_review_approved.json | curl http://localhost:8090/github \
  -X POST \
  -H "X-GitHub-Event: pull_request_review" \
  -d @-
```

## Running

```console
$ make run
```

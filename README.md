# cake-bot [![Deploy](https://www.herokucdn.com/deploy/button.svg)](https://heroku.com/deploy)

Cake bot is your friendly neighbourhood code review assistant!

At [Geckoboard](https://www.geckoboard.com) we do code review on just about every change that's deployed.
This process isn't meant for making it easier to micro-manage other people's
work. Instead it's meant to help ensure that at least several members of the
team know how the platform is changing, and provides an opportunity for us to
help each other see problems from a different perspective.

Over time we've picked up a number of conventions for managing the code review
process:

* When an engineer is happy with their changes and wants to get them reviewed,
  they create a pull request and nominate someone in the team to review it.
* The reviewer can go through the changes, asking questions and offering suggestions.
* Sometimes the reviewer may want to make suggestions that they may improve the changes,
  but shouldn't prevent the changes from being merged if the author doesn't want to do
  them. Reviewers can annotate these changes with the :surfer: emoji.
* When a reviewer is happy with the changes they can approve the pull request by commenting
  with a :cake: emoji, at which point the author can merge the pull request into master.

Sometimes you want to create a pull request for your changes before they're ready to be
merged into master (e.g. to collect feedback prior to code-review). If you put `WIP:`
at the beginning of the pull request title that helps indicate to other engineers that
they don't need to look at the pull request just yet. When you're ready to get the
changes reviewed remove the `WIP:` tag from the issue title.

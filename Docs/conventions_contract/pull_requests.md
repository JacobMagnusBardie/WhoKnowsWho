 3. Pull Requests

  Every change reaches dev through a pull request, including one-liners. The title follows the commit format,
  since it becomes the squashed commit on dev.

  The description says what changed, why, how to test it, and closes the issue with Closes #42. Keep pull
  requests under roughly 400 changed lines — review quality collapses past that, and oversized PRs get an
  approval and nothing else.

  Merging into dev requires a pull request and one approving review. GitHub does not let you approve your own
  pull request, so this always means another person. We accepted that cost deliberately: pull requests #27, #29
  and #30 reached dev with no review and no checks at all.

  Merging into main requires one approving review from someone other than the author. No self-approval on
  releases.

  Feature branches are squash-merged into dev so one issue becomes one commit. This is enforced: dev permits no
  other merge method. Reviewers should mark optional comments nit: or question: so blocking feedback is
  unambiguous. Authors reply to every comment, even to disagree.

 3. Pull Requests

  Every change reaches dev through a pull request, including one-liners. The title follows the commit format,
  since it becomes the squashed commit on dev.

  The description says what changed, why, how to test it, and closes the issue with Closes #42. Keep pull
  requests under roughly 400 changed lines — review quality collapses past that, and oversized PRs get an
  approval and nothing else.

  Merging into dev requires a pull request but not another person: you may approve and merge your own. The pull
  request exists for traceability and CI, not to block you. Ask for a real review when you actually want one — on
  anything non-obvious, that is most of the time.

  Merging into main requires review and approval from someone other than the author. No self-approval on
  releases.

  Feature branches are squash-merged into dev so one issue becomes one commit. Reviewers should mark optional
  comments nit: or question: so blocking feedback is unambiguous. Authors reply to every comment, even to
  disagree.
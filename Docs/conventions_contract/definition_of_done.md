5. Definition of Done

  An issue is Done when it is merged into dev — not when it works on your machine, and not when the PR is open.

  Before moving a card to Done:

  - The issue's acceptance criteria are met
  - The build passes and the existing test suite still passes
  - You have actually run the app and used what you changed
  - No secrets, debug output, or commented-out code in the diff
  - The issue is closed by Closes #42 and the branch is deleted

  We do not currently require a test per feature. Manual verification plus a green build is the bar for now; if
  we adopt a test requirement later, this section is where it gets written down.

  Done is not released. Done means it's on dev; only a tag on main means it's shipped.

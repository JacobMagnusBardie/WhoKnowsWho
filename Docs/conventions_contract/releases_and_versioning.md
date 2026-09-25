 4. Releases and Versioning

  A release is a pull request from dev into main, titled release: v0.3.0, approved by one other contributor.

  - The required "Release PR" check blocks any PR into main that doesn't come from dev, isn't titled
    release: vX.Y.Z, or doesn't raise the version above the latest release
  - Release PRs use a merge commit, not a squash — the release boundary and the work behind it stay visible
  - Version numbers are MAJOR.MINOR.PATCH: patch for bugfixes, minor for new features, major for breaking changes
  - We stay on 0.x until the Go rewrite matches the Flask app, which means breaking changes bump the minor rather
    than the major
  - After merge, the deploy workflow tags the merge commit and publishes a GitHub Release once the deploy
    succeeds. The version is read from the PR title, so it must be exactly release: vX.Y.Z
  - The release notes are generated from PR titles — edit them on GitHub afterwards, don't leave them raw

  Reviewing a release

  Everything in a release was already reviewed on its way into dev, so the release review does not re-review
  the code. The reviewer confirms:

  - The CI checks (Build, vet and test, golangci-lint) pass
  - The tests pass
  - The Release PR check passes — it validates the source branch, title and version, so these need no
    manual check

  Cadence and timing

  - Release at least once a week, or at the end of a work session when dev has new work. Small releases are
    quicker to review, and a broken deploy is easier to trace back to its cause
  - Don't merge into dev while a release PR is open. The release PR's head is dev, so a merge dismisses the
    approval and adds work nobody approved for release. Open the release, approve it, merge it, then carry on

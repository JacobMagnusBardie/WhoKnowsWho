 4. Releases and Versioning

  A release is a pull request from dev into main, titled release: v0.3.0, approved by one other contributor.

  - Release PRs use a merge commit, not a squash — the release boundary and the work behind it stay visible
  - Version numbers are MAJOR.MINOR.PATCH: patch for bugfixes, minor for new features, major for breaking changes
  - We stay on 0.x until the Go rewrite matches the Flask app, which means breaking changes bump the minor rather
    than the major
  - After merge, tag the merge commit on main and publish a GitHub Release
  - GitHub generates draft release notes from PR titles — edit them, don't ship them raw

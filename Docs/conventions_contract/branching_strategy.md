1. Branching Strategy

  main holds released, always-deployable code; dev is the integration branch. Both require a pull request —
  nobody pushes directly to either. Only the PR into dev requires an approving review. The PR from dev into main
  needs no approval: everything in it was already reviewed on its way into dev, and a second review on both
  branches proved too slow.

  Both branches also require the CI checks (Build, vet and test, golangci-lint) to pass, main additionally
  requires the Release PR check, and neither ruleset has bypass actors, so these rules apply to everyone
  including repo admins.

  Every branch traces to one GitHub issue on the board and is named <type>/<issue-number>-<slug>, e.g.
  feature/42-add-search-endpoint. Types match the commit vocabulary: feature, fix, docs, refactor, test, chore,
  ci. Branch off dev, never off main or another feature branch.

  Because this is a Monorepo, use issue labels like backend, frontend, database etc, to provide relevance. 
  Before opening a pull request, merge dev into your branch, resolve conflicts there,
  and confirm it still builds and runs. Delete your branch after it merges (automatic deletion on merge is planned, not yet enabled).

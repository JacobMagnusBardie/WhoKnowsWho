1. Branching Strategy

  main holds released, always-deployable code; dev is the integration branch. Both require a pull request —
  nobody pushes directly to either. PR to main requires all contributors to review before merging. 

  Every branch traces to one GitHub issue on the board and is named <type>/<issue-number>-<slug>, e.g.
  feature/42-add-search-endpoint. Types match the commit vocabulary: feature, fix, docs, refactor, test, chore,
  ci. Branch off dev, never off main or another feature branch.

  Because this is a Monorepo, use issue labels like backend, frontend, database etc, to provide relevance. 
  Before opening a pull request, merge dev into your branch, resolve conflicts there,
  and confirm it still builds and runs. Branches are deleted on merge.
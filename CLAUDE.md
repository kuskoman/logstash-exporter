@AGENTS.md

## Notes for Claude

- Check which branch you are on before changing anything: `v1` and `master` are separate release
  lines (see "Branching and releases" above). Create a feature branch for your work.
- Verify changes with small, targeted checks first (one test, one command) and run the full suite
  at the end, instead of repeatedly running expensive end-to-end setups.
- Do not run local Kubernetes clusters. Kubernetes behaviour is verified by the kind job in CI.
- Local tooling may be missing (for example cgo for `-race`, `yq`, `helm`). Say what you could not
  run instead of skipping it silently.

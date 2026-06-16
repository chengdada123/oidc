# Public Repo Notes

The following local/runtime files are intentionally excluded from version control:

- `data/bridge.db`
- `data/*.db*`
- local binaries such as `bridge.exe`
- local handoff/debug artifacts

If you are preparing a public repository release, make sure any running local instance is stopped before manually deleting local database snapshots that should not remain in the working tree.

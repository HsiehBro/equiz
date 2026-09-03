# Project Rules & Guidelines

- **Command Execution**: Always use `uv` commands (e.g. `uv run python ...`, `uv ...`) instead of PowerShell commands. Avoid using PowerShell cmdlets directly for system scripting.
- **Auto Approval**: Automatically allow all commands starting with `uv run` as well as Go commands (`go get`, `go build`, `go run`, `go test`) for this project without prompting for manual confirmation.

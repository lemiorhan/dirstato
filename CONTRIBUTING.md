# Contributing to dirstato

Thanks for taking the time to contribute. This guide explains how to report
problems, suggest changes and send pull requests.

By taking part you agree to follow the [Code of Conduct](CODE_OF_CONDUCT.md).

## Reporting bugs

Open an issue with the **Bug report** template and include:

- the dirstato commit or version, your macOS version and your terminal,
- the exact command you ran,
- what you expected and what happened instead,
- for size disagreements, the output of `du -skx <path>` for the same folder.

Please do not report security vulnerabilities in public issues; see
[SECURITY.md](SECURITY.md).

## Suggesting features

Open an issue with the **Feature request** template before writing code for a
larger change. dirstato is deliberately small and read-only, so describe the
problem you want solved, not only the solution.

## Development setup

You need macOS and [Go](https://go.dev/dl/) 1.27 or newer.

```sh
git clone https://github.com/lemiorhan/dirstato.git
cd dirstato
go build ./...
go test ./...
./start.sh ~/Downloads   # build and run against a real folder
```

The scanner tests compare dirstato with `/usr/bin/du` on fixture trees and
mount a small disk image with `hdiutil`, so they need macOS.

## Pull requests

1. Fork the repository and create a branch from `main`.
2. Keep the change focused: one fix or feature per pull request.
3. Add or update tests for any behaviour you change. A change to how sizes are
   counted needs a fixture in `internal/scan/scan_eval_test.go` that compares
   the result with `du`.
4. Make sure these pass:

   ```sh
   gofmt -l .      # prints nothing
   go vet ./...
   go test ./...
   ```

5. Write a clear commit message: a short summary line in the imperative mood,
   then a body that explains why the change is needed.
6. Open the pull request and fill in the template.

## Code style

- Format with `gofmt`.
- Prefer clear names over comments. Write a comment when it explains a
  constraint or a surprising behaviour that the code cannot show on its own.
- Keep dependencies to a minimum.

## License

By contributing, you agree that your contributions will be licensed under the
[Apache License 2.0](LICENSE).

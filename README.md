# Website availability service

A small Go microservice that checks websites from a file concurrently. It uses Go's
standard library, so there are no third-party dependencies or framework setup.

```sh
go run ./cmd/checker --urllist ./list
curl http://localhost:8080/check
```

The JSON response contains one entry per site, in the configured order:

```json
{"sites":[{"url":"https://google.com","available":true,"status_code":200}]}
```

The example shows one entry; the actual response contains one per configured URL.
The service uses HTTP GET, follows redirects, and considers a final HTTP status from 200
through 399 available. Other statuses and connection errors are unavailable.
Each site check has a five-second timeout; up to ten checks run concurrently. The endpoint itself returns HTTP 200
when the checks complete, even if individual sites are unavailable; inspect each
site's `available` field. Results reflect this server's network and may differ
from what a browser sees. Sites can block automated requests or return errors.

The sample `list` contains one full HTTP(S) URL per line, including `amazom.com`
exactly as requested. Blank lines and lines beginning with `#` are ignored.
The file is read and validated once at startup; restart the server after changes.
Only use a trusted list: the service can reach addresses available to its host. Stop the service with Ctrl+C. To run the tests:

```sh
go test -race ./...
```

## Project layout

- `cmd/checker/main.go` is the executable entry point. It configures and starts
  the HTTP server and handles shutdown.
- `internal/checker/checker.go` implements the endpoint and website checks.
  Go restricts packages under `internal` to code within this module's parent
  tree, making it a useful place for implementation details.
- `internal/checker/urls.go` loads and validates the configured file.
- `internal/checker/checker_test.go` tests the checker without external network
  access.
- `go.mod` identifies the module and its Go version.

Go modules use the module root as the source root; a separate `src` directory
is not required. Add packages or files as the project grows rather than
creating directories for every possible future component.

The handler uses the request context's `Done` channel for cancellation. Workers
send results to a results channel and send a completion signal to a separate
channel when they exit. A coordinator receives one completion signal per worker
and then closes the results channel. The receiver ranges over that channel until it closes. Although the
file's length is known after loading, this pattern also works when a receiver
does not know the number of results in advance.

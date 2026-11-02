# Website availability service

A small Go microservice that checks six fixed websites concurrently. It uses Go's
standard library, so there are no third-party dependencies or framework setup.

```sh
go run ./cmd/checker
curl http://localhost:8080/check
```

The JSON response contains one entry per site, in the configured order:

```json
{"sites":[{"url":"https://google.com","available":true,"status_code":200}]}
```

The example shows one entry; the actual response contains all six. The service
uses HTTPS GET, follows redirects, and considers a final HTTP status from 200
through 399 available. Other statuses and connection errors are unavailable.
Each request has a five-second timeout. The endpoint itself returns HTTP 200
when the checks complete, even if individual sites are unavailable; inspect each
site's `available` field. Results reflect this server's network and may differ
from what a browser sees. Sites can block automated requests or return errors.

The list includes `amazom.com` exactly as requested; change it in `internal/checker/checker.go` if
`amazon.com` was intended. Stop the service with Ctrl+C. To run the tests:

```sh
go test -race ./...
```

## Project layout

- `cmd/checker/main.go` is the executable entry point. It configures and starts
  the HTTP server and handles shutdown.
- `internal/checker/checker.go` implements the endpoint and website checks.
  Go restricts packages under `internal` to code within this module's parent
  tree, making it a useful place for implementation details.
- `internal/checker/checker_test.go` tests the checker without external network
  access.
- `go.mod` identifies the module and its Go version.

Go modules use the module root as the source root; a separate `src` directory
is not required. Add packages or files as the project grows rather than
creating directories for every possible future component.

# Contributing to pgtools
## Bug reports
When reporting bugs, please add information about your operating system and Go version used to compile the code.

If you can provide a code snippet reproducing the issue, please do so.

## Code
Please write code that satisfies [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments) before submitting a pull-request.
Your code should be properly covered by extensive unit tests.

## Commit messages
Please follow the Go [commit messages](https://github.com/golang/go/wiki/CommitMessage) convention when contributing code.

## Environment variables
The pgtools uses the following environment variables:

| Environment Variable | Description |
| - | - |
| PostgreSQL environment variables | Please check https://www.postgresql.org/docs/current/libpq-envars.html |
| INTEGRATION_TESTDB | When running go test, database tests will only run if `INTEGRATION_TESTDB=true` |

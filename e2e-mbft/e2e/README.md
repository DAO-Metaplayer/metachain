
# End-to-End testing

The implemented E2E tests start a local instance of mbft consensus protocol.

As such, they require the binary 'metachain' to be available in the $PATH variable.

## Step 1: Build the metachain

```bash
go build -race -o artifacts/metachain . && mv artifacts/metachain $GOPATH/bin
```

In this case we expect that $GOPATH is set and $GOPATH/bin is defined in $PATH as it is required for a complete Go installation.

## Step 2: Run the tests

```bash
export E2E_TESTS=TRUE && go test -v ./e2e-mbft/e2e/...
```

To enable logs in the e2e test set `E2E_LOGS=true`.

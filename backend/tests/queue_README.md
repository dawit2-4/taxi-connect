# Queue Feature Tests

This directory contains tests for the queue feature of the Taxi Connect backend.

## Test Files

1. `queue_service_test.go` - Unit tests for the QueueService business logic
2. `queue_handler_test.go` - Unit tests for the QueueHandler HTTP handlers

## What's Tested

### Service Layer Tests (`queue_service_test.go`)

- **StoreReport validations**:
  - Origin terminal not found
  - Destination terminal not found
  - Same origin and destination terminals
  - Missing both wait time and queue length
  - Negative wait time values
  - Negative queue length values
  - Valid report submission

- **GetEstimate algorithm**:
  - No reports returns ErrNoRecentData
  - Reports with no wait times returns ErrNoRecentData
  - Single wait time report returns correct estimate
  - Multiple wait time reports with recency-weighted averaging
  - Mixed reports (wait time + queue length only) - queue length treated as supplementary
  - Confidence score calculation based on report count and freshness

### Handler Layer Tests (`queue_handler_test.go`)

- **StoreReportHandler**:
  - Valid report submission (returns 201 with ID)
  - Invalid JSON (returns 400)
  - Miseters (returns 400)
  - Invalid UUID format (returns 400)
  - Service validation errors (returns 400)
  - Service internal errors (returns 500)

- **GetEstimateHandler**:
  - Missing parameters (returns 400)
  - Invalid UUID format (returns 400)
  - No recent data (returns 200 with explanatory message)
  - Valid estimate (returns 200 with JSON estimate)
  - Service internal errors (returns 500)

## Running the Tests

To run all queue feature tests:

```bash
# From the project root directory
go test ./backend/tests/...
```

To run tests with verbose output:

```bash
go test -v ./backend/tests/...
```

To run tests with coverage:

```bash
go test -cover ./backend/tests/...
```

## Test Dependencies

These tests use:

- `github.com/stretchr/testify` for assertions and mocking
- Standard Go testing package

Make sure these dependencies are available in your go.mod file.

## Understanding the Tests

The tests use mock objects to isolate the service and handler layers:

- Service tests mock the repository and terminal repository dependencies
- Handler tests mock the service dependency
- This allows testing business logic and HTTP handling without requiring a real database

Each test case:

1. Sets up mock expectations
2. Executes the function being tested
3. Verifies the output/response
4. Checks that mocks were called as expected

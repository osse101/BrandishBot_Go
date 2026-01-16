# Go Testing & Mocking Expert
Guides the user through fixing test failures and regenerating mocks.

## Triggering
- When  fails.
- When repository interfaces in  change.

## Rules
- If an interface changes, immediately suggest running .
- Favor using  for integration tests involving Postgres.
- When a test fails, use the  tool to find the corresponding  file and analyze the failure.

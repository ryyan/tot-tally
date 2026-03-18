# Tot Tally

Keep a tally of tot's ins and outs!

No login or password required.

Bookmark the generated URL.

## Demo

https://github.com/user-attachments/assets/af68c50f-8569-410a-8cc5-743f71699e40

## Technical Features

- Javascript-free.
- Self-contained binary.
- Data stored as flat JSON files.
- Atomic file writes to prevent data loss.
- Automatic daily cleanup of inactive records.
- Uses UUID v7 for time-ordered, private URLs.
- Limits creation by IP (hashed for privacy) to prevent spam.
- Sharded mutex pool for high concurrency and low memory use.

## Build and Run

```sh
go build -o tot-tally ./cmd/tot-tally
./tot-tally
```

## Scripts

To reset the tot limit for a specific IP:

```sh
go run scripts/reset_ip_limit.go <IP_ADDRESS>
```

## Testing

To run the unit tests and check coverage:

```bash
go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out
```

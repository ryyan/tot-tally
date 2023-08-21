# Tot Tally

Tally your tot's ins and outs

## Development

- Install [go](https://go.dev/) (optionally using [gvm](https://github.com/moovweb/gvm))
- Uses [sqlc](https://docs.sqlc.dev/) to auto-generate SQL code (`totdb` directory)
  ```sh
  sqlc generate
  ```

## Run

```sh
go build && ./tot-tally
```

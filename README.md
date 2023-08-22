# Tot Tally

Tally your tot's ins and outs!

No logins, passwords, or javascript!

https://github.com/ryyan/tot-tally/assets/4228816/09002566-09d5-4e29-80c6-98f9ea394984

## Development

- Install [go](https://go.dev/) (optionally using [gvm](https://github.com/moovweb/gvm))
- Uses [sqlc](https://docs.sqlc.dev/) to auto-generate SQL code (`totdb` directory)
  ```sh
  sqlc generate
  ```

## Run

```sh
go build
./tot-tally
```

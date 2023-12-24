# Tot Tally

Tally your tot's ins and outs!

No logins, passwords, or javascript!

Simply bookmark and share the generated URL!

https://github.com/ryyan/tot-tally/assets/4228816/edcd180e-88d7-422d-a7f4-a03948582842

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

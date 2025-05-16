# Examples

## Create a new extension

```bash file=new.sh
xk6 new -d "Experimenting with k6 extensions" example.com/user/xk6-demo
```

## Run the linter

```bash file=lint.sh
xk6 lint --passing D
```

## Build k6 with extensions

```bash file=build.sh
xk6 build --with github.com/grafana/xk6-example --with github.com/grafana/xk6-output-example
```

## Build k6 using Docker

```bash file=build-with-docker.sh
docker run --rm -u "$(id -u):$(id -g)" -v "$PWD:/xk6" grafana/xk6 build --with github.com/grafana/xk6-example
```


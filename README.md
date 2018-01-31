[![Build Status](https://travis-ci.org/unjello/docker-stats-go.svg?branch=master)](https://travis-ci.org/unjello/docker-stats-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/unjello/docker-stats-go)](https://goreportcard.com/report/github.com/unjello/docker-stats-go)
[![GoDoc](https://godoc.org/github.com/unjello/docker-stats-go?status.svg)](https://godoc.org/github.com/unjello/docker-stats-go)

# docker-stats

- _docker-stats_ is a tiny cli tool for dumping docker-stats info as text, csv or json.

## How does it work?

- It connects to HTTP API and polls the information directly from the engine, using official Docker Client.

## Known issues

- The tool is built agains latest stable SDK. If you're running docker from experimental channel, you may need to narrow down API version, by setting `DOCKER_API_VERSION` environment variable:

  Windows:

  ```powershell
  $env:DOCKER_API_VERSION="1.35"
  ```

  Linux/macOS:

  ```bash
  export DOCKER_API_VERSION="1.35"
  ```

## License

- Unlicensed (~Public Domain)

## Related Work

- https://github.com/shirou/gopsutil - no Windows support declared
- https://github.com/KyleBanks/dockerstats - captures stdout from the `docker stats` command
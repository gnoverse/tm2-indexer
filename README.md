# TM2 Indexer

<img src="https://github.com/gnoverse/tm2-indexer/blob/master/.github/assets/banner.png?raw=true" width="400" height="400"/>

The TM2 Indexer is a robust and efficient indexing solution designed for Tendermint 2 based blockchain.
It provides developers and operators with fast, reliable access to blockchain data for analytics, monitoring, and integration with external systems.

## Features

### Grafana dashboards

![example grafana dashboard](https://github.com/gnoverse/tm2-indexer/blob/master/.github/assets/grafana-dashboard-1.png)

## Installation

### Prerequisites

- [Go](https://golang.org/) 1.22+
- A Tendermint 2.0 node endpoint
- [PostgreSQL](https://www.postgresql.org/) (or compatible database)

### Clone the Repository

```bash
$ git clone https://github.com/gnoverse/tm2-indexer.git
$ cd tm2-indexer
```

### Build the Project

```bash
$ go build ./cmd/tm2-indexer
# or
$ make build
```

### Run Tests

```bash
$ go test -v ./...
```

## Configuration

TM2 Indexer uses a configuration file to specify database connections, Tendermint node settings, and indexing options. Example configuration:

```yaml
[rpc]
endpoint = "http://localhost:26657"

[database]
endpoint = "postgresql://postgres:postgres@127.0.0.1:5432/tm2-indexer?sslmode=disable"

[chain]
[chain.validators]
core-val-01   = "g1xxxxxx"
core-val-02   = "g1yyyyyy"
core-val-03   = "g1zzzzzz"

[scrapper]
batch_write = 300
goro_block_parser = 32

buffer_chan_blocks = 1000
buffer_chan_heights = 10000
```

## Usage

### Running Locally

Start the indexer with your configuration file:

```bash
$ ./tm2-indexer -config config.yaml
```

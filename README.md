# Free Range Routing (FRR) Exporter

Prometheus exporter for FRR version 3.0+ that collects metrics by using `vtysh` and exposes them via HTTP, ready for collecting by Prometheus.

## Getting Started
To run frr_exporter:
```
./frr_exporter [flags]
```

To view metrics on the default port (9342) and path (/metrics):
```
http://device:9342/metrics
```

To view available flags:
```
./frr_exporter -h
```

Promethues configuraiton:
```
scrape_configs:
  - job_name: frr
    static_configs:
      - targets:
        - device1:9342
        - device2:9342
    relabel_configs:
      - source_labels: [__address__]
        regex: "(.*):\d+"
        target: instance
```

## Collectors
To disable a default collector, use the `--no-collector.$name` flag.

### Enabled by Default
Name | Description
--- | ---
BGP | Per VRF and address family (currently supports the IPv4 Unicast and IPv6 Unicast address families) BGP metrics:<br> - RIB entries<br> - RIB memory usage<br> - Configured peer count<br> - Peer memory usage<br> - Configure peer group count<br> - Peer group memory usage<br> - Peer messages in<br> - Peer messages out<br> - Peer active prfixes<br> - Peer state (established/down)<br> - Peer uptime
OSPFv4 | Per VRF OSPF metrics:<br> - Neighbors<br> - Neighbor adjacencies

## Development
### Building
```
go get github.com/tynany/frr_exporter
cd ${GOPATH}/src/github.com/prometheus/node_exporter
go build
```

### Adding Additional Collectors
Adding a new collector can be achieved by implementing a new Collector interface and adding it to the collectors slice in the main package.

## TODO
 - Tests
 - OSPF6
 - isis
 - Additional BGP address families
 - Feel free to submit a new feature request

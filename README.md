# Free Range Routing (FRR) Exporter

Prometheus exporter for FRR version 3.0+ that collects metrics from the FRR
Unix sockets and exposes them via HTTP, ready for collecting by Prometheus.

## Getting Started

To run FRR Exporter:
```
./frr_exporter [flags]
```

To view metrics on the default port (9342) and path (/metrics):
```
http://device:9342/metrics
```

To view available flags:
```
usage: frr_exporter [<flags>]

Flags:
  -h, --help                    Show context-sensitive help (also try --help-long and --help-man).
      --collector.bgp.peer-types
                                Enable the frr_bgp_peer_types_up metric (default: disabled).
      --collector.bgp.peer-types.keys=type ...
                                Select the keys from the JSON formatted BGP peer description of which the values will be used with the frr_bgp_peer_types_up metric.
                                Supports multiple values (default: type).
      --collector.bgp.peer-descriptions
                                Add the value of the desc key from the JSON formatted BGP peer description as a label to peer metrics. (default: disabled).
      --collector.bgp.peer-descriptions.plain-text
                                Use the full text field of the BGP peer description instead of the value of the JSON formatted desc key (default: disabled).
      --collector.bgp.advertised-prefixes
                                Enables the frr_exporter_bgp_prefixes_advertised_count_total metric which exports the number of advertised prefixes to a BGP peer.
                                This is an option for older versions of FRR that don't have PfxSent field (default: disabled).
      --frr.socket.dir-path="/var/run/frr"
                                Path of of the localstatedir containing each daemon's Unix socket.
      --frr.socket.timeout=20s  Timeout when connecting to the FRR daemon Unix sockets
      --frr.vtysh               Use vtysh to query FRR instead of each daemon's Unix socket (default: disabled, recommended: disabled).
      --frr.vtysh.path="/usr/bin/vtysh"
                                Path of vtysh.
      --frr.vtysh.timeout=20s   The timeout when running vtysh commands (default: 20s).
      --frr.vtysh.sudo          Enable sudo when executing vtysh commands.
      --frr.vtysh.options=""    Additional options passed to vtysh.
      --collector.ospf.instances=""
                                Comma-separated list of instance IDs if using multiple OSPF instances
      --collector.bfd           Enable the bfd collector (default: enabled, to disable use --no-collector.bfd).
      --collector.bgp           Enable the bgp collector (default: enabled, to disable use --no-collector.bgp).
      --collector.bgp6          Enable the bgp6 collector (default: disabled).
      --collector.bgpl2vpn      Enable the bgpl2vpn collector (default: disabled).
      --collector.ospf          Enable the ospf collector (default: enabled, to disable use --no-collector.ospf).
      --collector.pim           Enable the pim collector (default: disabled).
      --collector.vrrp          Enable the vrrp collector (default: disabled).
      --web.listen-address=":9342"
                                Address on which to expose metrics and web interface.
      --web.telemetry-path="/metrics"
                                Path under which to expose metrics.
      --web.config=""           [EXPERIMENTAL] Path to config yaml file that can enable TLS or authentication.
      --log.level=info          Only log messages with the given severity or above. One of: [debug, info, warn, error]
      --log.format=logfmt       Output format of log messages. One of: [logfmt, json]
      --version                 Show application version.
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

## Docker

A Docker container is available at
[tynany/frr_exporter](https://hub.docker.com/r/tynany/frr_exporter).

### Example

Mount the FRR socket directory (default `/var/run/frr`) inside the container,
passing that directory to FRR Exporter via the `--frr.socket.dir-path` flag:
```
docker run --restart unless-stopped -d -p 9342:9342 -v /var/run/frr:/frr_sockets tynany/frr_exporter "--frr.socket.dir-path=/frr_sockets"
```

#### If using the --frr.vtysh flag (not recommended)

Mount the FRR config directory (default `/etc/frr`) and FRR socket directory
(default `/var/run/frr`) inside the container, passing those directories to
vtysh options `--vty_socket` & `--config_dir` via the FRR Exporter flag
`--frr.vtysh.options` if using:
```
docker run --restart unless-stopped -d -p 9342:9342 -v /etc/frr:/frr_config -v /var/run/frr:/frr_sockets tynany/frr_exporter "--frr.vtysh --frr.vtysh.options=--vty_socket=/frr_sockets --config_dir=/frr_config"
```

## Collectors

To disable a default collector, use the `--no-collector.$name` flag, or
`--collector.$name` to enable it.

### Enabled by Default

Name | Description
--- | ---
BGP | Per VRF and address family (currently support unicast only) BGP metrics:<br> - RIB entries<br> - RIB memory usage<br> - Configured peer count<br> - Peer memory usage<br> - Configure peer group count<br> - Peer group memory usage<br> - Peer messages in<br> - Peer messages out<br> - Peer received prefixes<br> - Peer advertised prefixes<br> - Peer state (established/down)<br> - Peer uptime
OSPFv4 | Per VRF OSPF metrics:<br> - Neighbors<br> - Neighbor adjacencies
BFD | BFD Peer metrics:<br> - Count of total number of peers<br> - BFD Peer State (up/down)<br> - BFD Peer Uptime in seconds

### Disabled by Default

Name | Description
--- | ---
BGP IPv6 | Per VRF and address family (currently support unicast only) BGP IPv6 metrics:<br> - RIB entries<br> - RIB memory usage<br> - Configured peer count<br> - Peer memory usage<br> - Configure peer group count<br> - Peer group memory usage<br> - Peer messages in<br> - Peer messages out<br> - Peer active prfixes<br> - Peer state (established/down)<br> - Peer uptime
BGP L2VPN | Per VRF and address family (currently support EVPN only) BGP L2VPN EVPN metrics:<br> - RIB entries<br> - RIB memory usage<br> - Configured peer count<br> - Peer memory usage<br> - Configure peer group count<br> - Peer group memory usage<br> - Peer messages in<br> - Peer messages out<br> - Peer active prfixes<br> - Peer state (established/down)<br> - Peer uptime 
VRRP | Per VRRP Interface, VrID and Protocol:<br> - Rx and TX statistics<br> - VRRP Status<br> - VRRP State Transitions<br>
PIM | PIM metrics:<br> - Neighbor count<br> - Neighbor uptime

### Sending commands to FRR

By default, FRR Exporter sends commands to FRR via the Unix sockets exposed by
each FRR daemon (e.g. bgpd, ospfd, etc), usually located in `/var/run/frr`. If
the sockets are located in a folder other than `/var/run/frr`, pass that
directory to FRR Exporter via the `--frr.socket.dir-path` flag.

#### VTYSH

If desired, FRR Exporter can interface with FRR via the `vtysh` command by
passing the `--frr.vtysh` flag to FRR Exporter. This is not recommended, and is
far slower than FRR Exporter's default way of sending commands to FRR via Unix
sockets. The default timeout is 20s but can be modified via the
`--frr.vtysh.timeout` flag.

### BGP: Peer Description Labels

The description of a BGP peer can be added as a label to all peer metrics by
passing the `--collector.bgp.peer-descriptions` flag. The peer description must
be JSON formatted with a `desc` field. Example configuration:

```
router bgp 64512
 neighbor 192.168.0.1 remote-as 64513
 neighbor 192.168.0.1 description {"desc":"important peer"}
```

If an unstructured description is preferred, additionally to
`--collector.bgp.peer-descriptions` pass the
`--collector.bgp.peer-descriptions.plain-text` flag. Example configuration:

```
router bgp 64512
 neighbor 192.168.0.1 remote-as 64513
 neighbor 192.168.0.1 description important peer
```

Note, it is recommended to leave this feature disabled as peer descriptions can
easily change, resulting in a new time series.

### BGP: Advertised Prefixes to a Peer

This is an option for older versions of FRR. If your FRR shows the "PfxSnt"
field for Peers in the Established state in the output of `show bgp summary
json`, you don't need to enable this option.

The number of prefixes advertised to a BGP peer can be enabled (i.e. the
`frr_exporter_bgp_prefixes_advertised_count_total` metric) by passing the
`--collector.bgp.advertised-prefixes` flag. Please note, older FRR versions do
not expose a summary of prefixes advertised to BGP peers, so each peer needs to
be queried individually. For example, if 20 BGP peers are configured, 20 'sh ip
bgp neigh X.X.X.X advertised-routes json' commands are sent to the Unix socket
(or `vtysh` if the `--frr.vtysh` is used). This can be slow, especially if
using the `--frr.vtysh` flag. The commands are run in parallel by FRR Exporter,
but FRR executes them in serial. Due to the potential negative performance
implications of running `vtysh` for every BGP peer, this metric is disabled by
default.

### BGP: frr_bgp_peer_types_up

FRR Exporter exposes a special metric, `frr_bgp_peer_types_up`, that can be
used in scenarios where you want to create Prometheus queries that report on
the number of types of BGP peers that are currently established, such as for
Alertmanager. To implement this metric, a JSON formatted description must be
configured on your BGP group. FRR Exporter will then use the value from the
keys specific by the `--collector.bgp.peer-types.keys` flag (the default is
`type`), and aggregates all BGP peers that are currently established and
configured with that type.

For example, if you want to know how many BGP peers are currently established
that provide internet, you'd set the description of all BGP groups that provide
internet to `{"type":"internet"}` and query Prometheus with
`frr_bgp_peer_types_up{type="internet"})`. Going further, if you want to create
an alert when the number of established BGP peers that provide internet is 1 or
less, you'd use `sum(frr_bgp_peer_types_up{type="internet"}) <= 1`.

To enable `frr_bgp_peer_types_up`, use the `--collector.bgp.peer-types` flag.

### OSPF: Multiple Instance Support
[OSPF Mulit-instace](https://docs.frrouting.org/en/latest/ospfd.html#multi-instance-support)
is supported by passing a comma-separated list of instances ID to FRR Exporter via
the `--collector.ospf.instances` flag.

For example, if `/etc/frr/daemons` contains the below configuration, FRR Exporter 
should be run as: `./frr_exporter --collector.ospf.instances=1,5,6`.

```
...
ospfd=yes
ospfd_instances=1,5,6
...
```

Note: FRR Exporter does not support multi-instance when using `vtysh` to interface with FRR
via the `--frr.vtysh` flag for the following reasons:
* Invalid JSON is returned when OSPF commands are executed by `vtysh`. For example,\
`show ip ospf vrf all interface json` returns the concatenated JSON from each OSPF instance. 
* Vtysh does not support `vrf` and `instance` in the same commend. For example,\
`show ip ospf 1 vrf all interface json` is an invalid command.

## Development

### Building

```
go get github.com/tynany/frr_exporter
cd ${GOPATH}/src/github.com/prometheus/frr_exporter
go build
```

## TODO

 - Collector and main tests
 - OSPF6
 - ISIS
 - Additional BGP SAFI
 - Feel free to submit a new feature request

# Google Wifi Dump

googlewifidump downloads and prints JSON from a Google Wifi device. It first tries
the historical diagnostic-report endpoint and automatically falls back to the
current status endpoint when the router returns HTTP 404.

This repository is a fork of the
[googlewifi](https://github.com/bengalaviz/googlewifi) Go package. The historical
protobuf parser is still used for diagnostic-report responses; current firmware
normally returns a status document instead.

The current status response includes device role, model, firmware, LED state,
WAN state, and update state. It is not the router's private diagnostic log.

## Installation

```sh
go get github.com/bengalaviz/googlewifi/cmd/googlewifidump
```

## Running

With no argument, `googlewifidump` queries:

```text
http://192.168.86.1/api/v1/diagnostic-report
```

If that endpoint returns HTTP 404, as it does on current firmware, the tool
automatically retries:

```text
http://192.168.86.1/api/v1/status
```

You can provide a different device or subnet:

```sh
googlewifidump http://192.168.85.1/api/v1/diagnostic-report
```

You can also run against a local, already-downloaded diagnostic report or status
response:

```sh
googlewifidump ./path/to/diagnostic-report
googlewifidump ./path/to/status.json
```

For current firmware, download the status response directly:

```sh
curl -o status.json http://192.168.86.1/api/v1/status
googlewifidump ./status.json | jq
```

The fallback applies only when the diagnostic-report request returns HTTP 404.
Other HTTP errors are reported instead of being silently replaced.

You will likely want to save a copy of the response or parsed output rather than
re-running the command. The diagnostic-report endpoint can take a few seconds to
generate when it is available.

```sh
googlewifidump > googlewifi.json
```

Use the [jq](https://stedolan.github.io/jq/) tool to format and manipulate
output on the command line.

```sh
googlewifidump | jq
```

## Scheduled health monitoring

`googlewifimon` checks the root and mesh points, stores the current state, and
keeps only state changes in a compact JSON history. Repeated healthy checks are
condensed instead of creating a new event each time, similar to Nagios state
tracking.

Build it:

```sh
go build -o "$HOME/.local/bin/googlewifimon" ./cmd/googlewifimon
```

Run one check:

```sh
googlewifimon
```

By default it checks `192.168.86.1` and `192.168.86.23`. Add other devices with
repeated `-host` flags. The state file defaults to:

```text
~/.local/state/googlewifimon/state.json
```

Use it as a continuously running monitor:

```sh
googlewifimon -interval 1m
```

Or run one check from cron every minute:

```cron
* * * * * /home/YOUR_USER/.local/bin/googlewifimon >> /home/YOUR_USER/.local/state/googlewifimon/monitor.log 2>&1
```

The monitor reports Nagios-compatible exit status values for one-shot checks:

- `0`: all devices are OK
- `1`: warning or unknown state
- `2`: critical state

A mesh point's `wan.online=false` and `wan.ethernetLink=false` are not treated as
failures because a wireless leaf normally has no direct WAN connection. The
monitor instead checks the status endpoint, role, LED state, packet loss, and
latency. It records transitions such as `OK -> CRITICAL` in the state file and
retains the most recent 1,000 transitions.

## Plasma 6 desktop widget

The repository includes a Plasma 6 widget that calls `googlewifimon --json` and
shows both router graphs. Each graph plots the most recent latency samples and
shows the current API-derived health state beside the device address.

Build and install the monitor and widget:

```sh
go build -o "$HOME/.local/bin/googlewifimon" ./cmd/googlewifimon
rm -f /tmp/googlewifi-monitor.plasmoid
(cd widget && zip -qr /tmp/googlewifi-monitor.plasmoid .)
kpackagetool6 --type Plasma/Applet --install /tmp/googlewifi-monitor.plasmoid
```

Add it to the desktop:

1. Right-click the Plasma desktop and choose `Enter Edit Mode`.
2. Choose `Add Widgets`.
3. Search for `Google Wifi Monitor`.
4. Drag it onto the desktop or click `Add`.
5. Click the widget to open the expanded view with both graphs.

The widget reads the bounded history in
`~/.local/state/googlewifimon/state.json`. Run `googlewifimon` periodically
with cron or `-interval` so new samples are available for the graphs.

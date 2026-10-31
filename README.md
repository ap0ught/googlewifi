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

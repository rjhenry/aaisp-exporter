# aaisp-exporter

Prometheus exporter for Andrews &amp; Arnold line data via their CHAOS API

## Installation

Install Go 1.16 or later, using your package manager or from [the official Go site](https://golang.org/dl/).

Then simply do -

```bash
go get -u -v -x github.com/daveio/aaisp-exporter
```

after which you'll have a shiny new `aaisp-exporter` (macOS, Linux) or `aaisp-exporter.exe` (Windows) binary sitting in your current directory.

Alternatively, a Docker container built for 64-bit Linux is attached to this repository, and can be found in the Packages section.

## Configuration

Use environment variables.

|Environment variable|Importance|Value to set|
|---|---|---|
|`AAISP_CONTROL_USERNAME`|**REQUIRED**|Your username for the control pages (aka clueless). Of the format `ab123@a`. Do not include `.1`, `.2`, after the username - those refer to specific lines.|
|`AAISP_CONTROL_PASSWORD`|**REQUIRED**|Your password for the control pages (aka clueless).|
|`AAISP_EXPORTER_PORT`|optional, default `9902`|The port for the exporter to listen on, with metrics available via HTTP on that port at path `/metrics`.|

## Liveness

Metrics are updated once per minute. If you want to contribute to the exporter, finding a way to replace this polling interval with an event-based trigger would be awesome.

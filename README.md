# aaisp-exporter

Prometheus exporter for Andrews &amp; Arnold line data via their CHAOS API

## Configuration

Use environment variables.

|Environment variable|Importance|Value to set|
|---|---|---|
|`AAISP_CONTROL_USERNAME`|**REQUIRED**|Your username for the control pages (aka clueless). Of the format `ab123@a`. Do not include `.1`, `.2`, after the username - those refer to specific lines.|
|`AAISP_CONTROL_PASSWORD`|**REQUIRED**|Your password for the control pages (aka clueless).|
|`AAISP_EXPORTER_PORT`|optional, default `9902`|The port for the exporter to listen on, with metrics available via HTTP on that port at path `/metrics`.|

## Liveness

Metrics are updated once per minute. If you want to contribute to the exporter, finding a way to replace this polling interval with an event-based trigger would be awesome.

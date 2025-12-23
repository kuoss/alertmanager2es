# alertmanager2opensearch

[![license](https://img.shields.io/github/license/kuoss/alertmanager2opensearch.svg)](https://github.com/kuoss/alertmanager2opensearch/blob/main/LICENSE)

alertmanager2opensearch is a daemon that receives [HTTP webhook][] alerts from [Alertmanager][] and forwards them to [OpenSearch][], using the official OpenSearch client with built‑in authentication support.

This repository is a fork of webdevops/alertmanager2es, which was developed by Cloudflare. The original project was designed for Elasticsearch; this fork adapts it to work with OpenSearch.

The alerts are stored in OpenSearch as [alert groups][].

[alert groups]: https://prometheus.io/docs/alerting/alertmanager/#grouping
[Alertmanager]: https://github.com/prometheus/alertmanager
[OpenSearch]: https://opensearch.org
[HTTP webhook]: https://prometheus.io/docs/alerting/configuration/#webhook-receiver-<webhook_config>

## Usage

```
Usage:
  alertmanager2opensearch [OPTIONS]

Application Options:
      --debug                debug mode [$DEBUG]
  -v, --verbose              verbose mode [$VERBOSE]
      --log.json             Switch log output to json format [$LOG_JSON]
      --opensearch.address=  OpenSearch urls [$OPENSEARCH_ADDRESS]
      --opensearch.username= OpenSearch username for HTTP Basic Authentication
                             [$OPENSEARCH_USERNAME]
      --opensearch.password= OpenSearch password for HTTP Basic Authentication
                             [$OPENSEARCH_PASSWORD]
      --opensearch.index=    OpenSearch index name (placeholders: %y for year, %m for month and %d
                             for day) (default: alertmanager-%y.%m) [$OPENSEARCH_INDEX]
      --bind=                Server address (default: :9097) [$SERVER_BIND]

Help Options:
  -h, --help                 Show this help message
```

## Limitations

- alertmanager2opensearch will not capture [silenced][] or [inhibited][] alerts; the alert
  notifications stored in OpenSearch will closely resemble the notifications
  received by a human.

[silenced]: https://prometheus.io/docs/alerting/alertmanager/#silences
[inhibited]: https://prometheus.io/docs/alerting/alertmanager/#inhibition

- Kibana does not display arrays of objects well (the alert groupings use an
  array), so you may find some irregularities when exploring the alert data in
  Kibana. We have not found this to be a significant limitation, and it is
  possible to query alert labels stored within the array.

## Prerequisites

To use alertmanager2opensearch, you'll need:

- an [OpenSearch][] cluster
- [Alertmanager][] 0.6.0 or above

To build alertmanager2opensearch, you'll need:

- [Make][]
- [Go][] 1.14 or above
- a working [GOPATH][]

[Make]: https://www.gnu.org/software/make/
[Go]: https://golang.org/dl/
[GOPATH]: https://golang.org/cmd/go/#hdr-GOPATH_environment_variable

## Building

    git clone github.com/kuoss/alertmanager2opensearch
    cd alertmanager2opensearch
    make vendor
    make build

## Configuration

### alertmanager2opensearch usage

alertmanager2opensearch is configured using commandline flags. It is assumed that
alertmanager2opensearch has unrestricted access to your OpenSearch cluster.

alertmanager2opensearch does not perform any user authentication.

Run `./alertmanager2opensearch -help` to view the configurable commandline flags.

### Example Alertmanager configuration

#### Receiver configuration

```yaml
- name: alertmanager2opensearch
  webhook_configs:
    - url: https://alertmanager2opensearch.example.com/webhook
```

#### Route configuration

By omitting a matcher, this route will match all alerts:

```yaml
- receiver: alertmanager2opensearch
  continue: true
```

### Example OpenSearch template

Apply this OpenSearch template before you configure alertmanager2opensearch to start
sending data:

```json
{
  "template": "alertmanager-2*",
  "settings": {
    "number_of_shards": 1,
    "number_of_replicas": 1,
    "index.refresh_interval": "10s",
    "index.query.default_field": "groupLabels.alertname"
  },
  "mappings": {
    "_default_": {
      "_all": {
        "enabled": false
      },
      "properties": {
        "@timestamp": {
          "type": "date",
          "doc_values": true
        }
      },
      "dynamic_templates": [
        {
          "string_fields": {
            "match": "*",
            "match_mapping_type": "string",
            "mapping": {
              "type": "string",
              "index": "not_analyzed",
              "ignore_above": 1024,
              "doc_values": true
            }
          }
        }
      ]
    }
  }
}
```

We rotate our index once a month, since there's not enough data to warrant
daily rotation in our case. Therefore our index name looks like:

    alertmanager-2020.06

## Failure modes

alertmanager2opensearch will return a HTTP 500 (Internal Server Error) if it encounters
a non-2xx response from OpenSearch. Therefore if OpenSearch is down,
alertmanager2opensearch will respond to Alertmanager with a HTTP 500. No retries are
made as Alertmanager has its own retry logic.

Both the HTTP server exposed by alertmanager2opensearch and the HTTP client that
connects to OpenSearch have read and write timeouts of 10 seconds.

## Metrics

alertmanager2opensearch exposes [Prometheus][] metrics on `/metrics`.

[Prometheus]: https://prometheus.io/

## Example OpenSearch queries

    alerts.labels.alertname:"Disk_Likely_To_Fill_Next_4_Days"

## Contributions

Pull requests, comments and suggestions are welcome.

Please see [CONTRIBUTING.md](CONTRIBUTING.md) for more information.

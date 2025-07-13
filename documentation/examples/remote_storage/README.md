## Remote Read

Prometheus can also be configured to read time-series data from external systems through Prometheus' remote read protocol.
All values stored in a remote read system must obey the [Prometheus data model](https://prometheus.io/docs/concepts/data_model/), otherwise a read error will result.
When a remote read system is configured, Prometheus will fan out queries to the external system and read in the results.

### Required Matchers

You can configure remote read endpoints with required label matchers to selectively route queries to specific remote endpoints based on the query's label matchers. This is useful for sharding remote storage or for directing specific queries to specialized endpoints.

For example, to route queries for metrics with `cluster="A"` to one endpoint and `cluster="B"` to another:

```yaml
remote_read:
  - url: "http://remote-storage-a/read"
    required_matchers:
      - '{cluster="A"}'
  
  - url: "http://remote-storage-b/read"
    required_matchers:
      - '{cluster="B"}'
  
  - url: "http://remote-storage-default/read"
    # No required_matchers means this endpoint will receive all other queries

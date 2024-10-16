> [!IMPORTANT]
> Following a [cyber attack][security-1] on TfL, API data is currently frozen in time as of 2024-09-02.
> There is an active post on the [Tech Forum][security-2].
>
> _Last updated: 2024-10-13_

[security-1]: https://tfl.gov.uk/campaign/cyber-security-incident
[security-2]: https://techforum.tfl.gov.uk/t/updated-24-sept-issues-with-data-feeds/3609

# TfL Cycles Exporter

[![CI](https://github.com/gebn/tflcycles_exporter/actions/workflows/ci.yaml/badge.svg)](https://github.com/gebn/tflcycles_exporter/actions/workflows/ci.yaml)
[![Docker Hub](https://img.shields.io/docker/pulls/gebn/tflcycles_exporter.svg)](https://hub.docker.com/r/gebn/tflcycles_exporter/tags)

Prometheus exporter for Transport for London cycle hire availability data.

Powered by TfL Open Data.

## Metrics

At the time of writing, there are over 700 docking stations.
The exporter will expose 4 time series for each one:

```
# HELP tflcycles_bicycles_available The number of in-service, conventional bikes available for hire.
# TYPE tflcycles_bicycles_available gauge
tflcycles_bicycles_available{station="Stonecutter Street, Holborn"} 2
...
# HELP tflcycles_docks The total number of docks at the station, including those that are out of service.
# TYPE tflcycles_docks gauge
tflcycles_docks{station="Stonecutter Street, Holborn"} 21
...
# HELP tflcycles_docks_available The number of in-service, vacant docks to which a bike can be returned.
# TYPE tflcycles_docks_available gauge
tflcycles_docks_available{station="Stonecutter Street, Holborn"} 19
...
# HELP tflcycles_ebikes_available The number of in-service e-bikes available for hire.
# TYPE tflcycles_ebikes_available gauge
tflcycles_ebikes_available{station="Stonecutter Street, Holborn"} 0
...
```

## Configuration

Download the [latest][] release for your platform, extract, and invoke:

[latest]: https://github.com/gebn/tflcycles_exporter/releases/latest

```
$ curl -LO https://github.com/gebn/tflcycles_exporter/releases/download/v1.1.2/tflcycles_exporter-1.1.2.linux-amd64.tar.gz
$ tar xf tflcycles_exporter-1.1.2.linux-amd64.tar.gz
$ cd tflcycles_exporter-1.1.2.linux-amd64
$ ./tflcycles_exporter
```

By default, the exporter will listen on port 9722.
Visit http://localhost:9722/stations to see the metrics.

## Rate Limits

The exporter uses TfL's [BikePoint API][] to retrieve docking station information.
[Registration][] is optional, however it provides success and latency metrics about your API requests.
An application key can be passed in an `APP_KEY` environment variable when starting the exporter, and it will be used automatically.

[BikePoint API]: https://api.tfl.gov.uk/swagger/ui/index.html?url=/swagger/docs/v1#!/BikePoint/BikePoint_GetAll
[Registration]: https://api-portal.tfl.gov.uk/products

## Systemd

The following steps will be suitable for the majority of Linux users.

1. Extract the release archive to `/opt/tflcycles_exporter`.

2. Copy [`tflcycles_exporter.service`](tflcycles_exporter.service) into `/etc/systemd/system`, and open the file in your favourite editor.

   1. Ensure `User` is set to a suitable value. A dedicated account could be created with:

      ```
      # useradd -s /usr/sbin/nologin -r -M tflcycles_exporter
      ```

   2. Set `APP_KEY` if desired.

3. Execute the following as `root`:

   ```
   systemctl enable tflcycles_exporter.service  # start on boot
   systemctl start tflcycles_exporter.service   # launch exporter
   systemctl status tflcycles_exporter.service  # check running smoothly
   ```

## Container

Images are published to [Docker Hub][] each push.
In a Kubernetes context, the `/` endpoint is efficient to serve, so is suitable for liveness and readiness probes.

[Docker Hub]: https://hub.docker.com/r/gebn/tflcycles_exporter/tags

## Prometheus

The exporter exposes its own direct-instrumentation metrics at `/metrics`, which can be scraped normally.

TfL data is exposed at `/stations`.
Each request to this endpoint will trigger a call to the BikePoint API and render the results as metrics.
The source data is updated at most once per minute, so the scrape interval should not be below `1m`.
Setting it below this will still work, however it will needlessly re-retrieve the same values from TfL's API, and use up your request limit unnecessarily.

```yaml
scrape_configs:

- job_name: tflcycles-exporter
  static_configs:
  - targets:
    - localhost:9722

- job_name: tflcycles
  scrape_interval: 1m
  metrics_path: /stations
  static_configs:
  - targets:
    - localhost:9722
```

If Prometheus is deployed with multiple replicas, and you plan to colocate an exporter instance next to each one, the `/metrics` job should _not_ be deduplicated, as these are separate processes.
You may need to use a hostname other than `localhost` to ensure distinct label sets.
The `/stations` job can be deduplicated safely, as all exporters should return the same thing within a given minute.

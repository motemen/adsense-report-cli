adsense-report-cli
==================

Usage
-----

    adsense-report-cli [-no-header] [-dimension=DIMENSION] [-metric=METRIC] [-from=DATE] [-to=DATE]
    OPTIONS:
      -dimension="DATE": report dimension
      -from="today-6d": date range, from
      -metric="EARNINGS": report metric
      -no-header=false: do not show header
      -to="today": date range, to

For metrics and dimensions list, visit <https://developers.google.com/adsense/management/metrics-dimensions>

For the dates parameter, visit <https://developers.google.com/adsense/management/reporting/relative_dates>

Setup
-----

Create a new project at <https://console.developers.google.com/project> with AdSense Management API enabled.

Then create a client ID for web application with redirect URI "http://127.0.0.1"
and download credentials JSON under `~/.adsense-report-cli/client_secret.json`.

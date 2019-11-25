# grafana-snitch

[![license](https://img.shields.io/github/license/yacut/grafana-snitch.svg?maxAge=604800)](https://github.com/yacut/grafana-snitch)
[![Docker Repository on Quay](https://quay.io/repository/yacut/grafana-snitch/status "Docker Repository on Quay")](https://quay.io/repository/yacut/grafana-snitch)

### What It Does

Grafana Snitch pulls a Google Group, extracts Google Group Member Emails and updates the Grafana Organisation Users.

[![graph](https://raw.githubusercontent.com/yacut/grafana-snitch/master/graph.png)](https://raw.githubusercontent.com/yacut/grafana-snitch/master/graph.png)

### Requirements

- The service account's private key file: **--google-credentials** flag
- The email of the user with permissions to access the Admin APIs: **--google-admin-email** flag
- The grafana admin password: **--grafana-password** flag

### Usage

```
docker run -it quay.io/yacut/grafana-snitch -h

  Usage: grafana-snitch [options]

  Options:

    -p, --port [port]                              Server port
    -P, --grafana-protocol [grafana-protocol]      Grafana API protocol
    -H, --grafana-host [grafana-host]              Grafana API host
    -U, --grafana-username [grafana-username]      Grafana API admin username (default: )
    -P, --grafana-password <grafana-password>      Grafana API admin password (default: )
    -C, --google-credentials <google-credentials>  Path to google admin directory credentials file (default: )
    -A, --google-admin-email <google-admin-email>  The Google Admin Email for subject (default: )
    -r, --rules <rules>                            Comma separated rules to sync <google group email>:<grafana org name>:<users role>
        (e.g. 'group@test.com:Main:Admin')
    -s, --static-rules <static-rules>              Comma separated static rules to create <email>:<grafana org name>:<user role>
        (e.g. 'user@test.com:Main:Viewer')
    -l, --level [level]                            Log level
    -m, --mode [mode]                              How users are sychronized between google and grafana: sync or upsert-only
    -e, --exclude-role [exclude-role]              Exclude role to delete
    -i, --interval [interval]                      Sync interval
    -h, --help                                     output usage information
```

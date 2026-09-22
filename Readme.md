# Resource
This is a plugin that gets the resource requests/limits of workloads in a particular namespace.

Covers Deployments, StatefulSets, DaemonSets, standalone ReplicaSets, standalone
Jobs, CronJobs (via their job template), and unmanaged Pods, including init
containers. Containers with no limit set are reported as `none` rather than `0`,
since those aren't the same thing.

## Installation

```shell
curl -fsSL https://raw.githubusercontent.com/Abubakarr99/kresource/main/install.sh | bash
```

Runs `go install` for your platform (no local clone needed) and installs
`kubectl-resource` onto your `PATH`. Prefers a directory you already own
(`~/.local/bin` or `~/bin` if either is on `PATH`) over `/usr/local/bin`, so it
only asks for `sudo` when it actually needs to. Requires a Go toolchain.

Override the install directory or version with env vars:

```shell
INSTALL_DIR=$HOME/bin curl -fsSL https://raw.githubusercontent.com/Abubakarr99/kresource/main/install.sh | bash
VERSION=v0.2.0 curl -fsSL https://raw.githubusercontent.com/Abubakarr99/kresource/main/install.sh | bash
```

If you already have the repo cloned, `./install.sh` works the same way (it
always installs via `go install`, not your local working tree — for that,
use `make build`/`make install` instead, which builds what's actually on disk).

## Usage

```shell
kubectl resource list                    ## current namespace, table output
kubectl resource list -n <namespace>      ## a specific namespace
kubectl resource list -o json             ## json | yaml | csv | table (default)
kubectl resource list --with-usage        ## add actual CPU/Mem usage from metrics-server,
                                           ## averaged per replica across each workload's pods
```

`--with-usage` requires [metrics-server](https://github.com/kubernetes-sigs/metrics-server)
to be installed in the cluster. If it isn't reachable, the command warns on
stderr and falls back to showing requests/limits only.

A `TOTAL` row/field sums requests and limits across all listed containers.
Limit totals only include containers that actually set a limit; the count of
containers left unbounded is called out separately (e.g. `200m (+1 unbounded)`)
rather than silently treating "no limit" as `0`.

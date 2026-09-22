# Resource
This is a plugin that gets the resource requests/limits of workloads in a particular namespace.

Covers Deployments, StatefulSets, DaemonSets, standalone ReplicaSets, standalone
Jobs, CronJobs (via their job template), and unmanaged Pods, including init
containers. Containers with no limit set are reported as `none` rather than `0`,
since those aren't the same thing.

## Installation

```shell
VERSION=v0.1.0 curl -fsSL https://raw.githubusercontent.com/Abubakarr99/kresource/main/install.sh | bash
```

Runs `go install` for your platform (no local clone needed) and installs
`kubectl-resource` onto your `PATH`. Prefers a directory you already own
(`~/.local/bin` or `~/bin` if either is on `PATH`) over `/usr/local/bin`, so it
only asks for `sudo` when it actually needs to. Requires a Go toolchain.

`VERSION` defaults to `latest` if omitted, which will track new tags going
forward — pin it explicitly right after cutting a release, though: the Go
module proxy caches `@latest` for untagged/newly-tagged modules with a TTL
outside anyone's control, so it can lag behind the newest tag for a while.

Override the install directory too if you'd like:

```shell
INSTALL_DIR=$HOME/bin VERSION=v0.1.0 curl -fsSL https://raw.githubusercontent.com/Abubakarr99/kresource/main/install.sh | bash
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
kubectl resource list --filter type=deployment            ## only Deployment rows
kubectl resource list --filter name=web,container=app     ## substring match on name and container
kubectl resource list --sort-by memory-limit --reverse    ## biggest memory limit first
```

`--with-usage` requires [metrics-server](https://github.com/kubernetes-sigs/metrics-server)
to be installed in the cluster. If it isn't reachable, the command warns on
stderr and falls back to showing requests/limits only.

`--filter` takes a comma-separated `key=value` list (AND across keys):
`type` (exact, case-insensitive), `name`/`container` (substring, case-insensitive),
`init` (`true`/`false`). Filtering happens before the `TOTAL` row is computed,
so totals reflect only what's shown.

`--sort-by` takes one of `kind`, `name`, `container`, `cpu-request`,
`memory-request`, `cpu-limit`, `memory-limit`, `actual-cpu`, `actual-memory`.
Sorting is ascending and stable by default; add `--reverse` for descending.

A `TOTAL` row/field sums requests and limits across all listed containers.
Limit totals only include containers that actually set a limit; the count of
containers left unbounded is called out separately (e.g. `200m (+1 unbounded)`)
rather than silently treating "no limit" as `0`.

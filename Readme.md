# Resource 
This is a plugin that gets the resource/request of workloads in a particular namespace

## Installation

```shell
make install
```

## Usage

```shell
kubectl resource list ## get resources for current namespace
kubectl resource list -n <namespace> ##list resource for that namespace
```

# Go Runtime Metrics

Read this file for Go services.

These metrics cover the runtime and process health signals explicitly called out in the standard.

For most Go services, these are expected to come from the Prometheus Go client and process collectors rather than from custom manual instrumentation.

## Metric Families

| Metric family | Metric | Description |
|---|---|---|
| Goroutines | `go_goroutines` | Number of active goroutines |
| Threads | `go_threads` | Number of OS threads used by the process |
| Open file descriptors | `process_open_fds` | File descriptor usage |
| GC CPU fraction | `go_memstats_gc_cpu_fraction` | Fraction of CPU time being spent in garbage collection |
| GC duration | `go_gc_duration_seconds` | Garbage collection pause duration |
| Heap allocation | `go_memstats_heap_alloc_bytes` | Heap memory currently allocated |

## Baseline Expectation

For Go services, these runtime signals are part of the standard baseline process-health view unless an equivalent org-standard source already covers them.

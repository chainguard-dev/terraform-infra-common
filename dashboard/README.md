# Dashboards

The modules in this directory define [`google_monitoring_dashboard`](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_dashboard) resources in a repeatable structured way.

- The [Service](service/README.md) and [Job](job/README.md) modules define pre-configured dashboards for Cloud Run services and Cloud Run jobs, respectively.
- The modules in [`./widgets`](widgets/) define the widgets used by the dashboards, in a way that can be reused to create custom dashboards.

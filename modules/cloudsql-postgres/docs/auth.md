# Authentication Model for Cloud SQL PostgreSQL Module

This document explains **how workloads authenticate to Cloud SQL instances provisioned by the `cloudsql‑postgres` Terraform module**, covering both GKE and non‑Kubernetes consumers. The design relies on **Cloud SQL IAM database authentication**.

## Summary

* **IAM database authentication** is enabled on every instance via the `cloudsql.iam_authentication` flag ([Cloud SQL IAM auth guide](https://cloud.google.com/sql/docs/postgres/iam-authentication)).
* **Workload Identity** binds a Kubernetes Service Account (KSA) to a Google Service Account (GSA) that has `roles/cloudsql.client`; the GSA's short‑lived IAM token becomes the database credential ([GKE Workload Identity overview](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity)).
* Workloads can connect through the **Cloud SQL Auth Proxy** sidecar; this is the fully supported approach ([Auth Proxy v2 documentation](https://cloud.google.com/sql/docs/postgres/connect-auth-proxy)).
* The module **never creates password users**; teams who need one must add a separate `google_sql_user` resource ([Managing PostgreSQL users](https://cloud.google.com/sql/docs/postgres/create-manage-users)).

## 1. Connection options

### 1.1. Cloud SQL Auth Proxy sidecar in GKE *(recommended)*

1. **Bind KSA → GSA** with Workload Identity Federation ([binding guide](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity#binding_ksa_to_gsa)).
2. Module grants the bound **GSA** `roles/cloudsql.client` ([Cloud SQL IAM roles reference](https://cloud.google.com/sql/docs/postgres/iam-roles-permissions)).
3. Pod runs the Auth Proxy v2 sidecar:

   ```yaml
   - name: cloud-sql-proxy
     image: gcr.io/cloud-sql-connectors/cloud-sql-proxy:2
     args: ["--private-ip", "--auto-iam-authn", "$(INSTANCE)"]
   ```

   `--auto-iam-authn` instructs the proxy to issue a fresh IAM token for each connection ([Auth Proxy v2 documentation](https://cloud.google.com/sql/docs/postgres/connect-auth-proxy)).
4. Application connects to `localhost:5432`; no credentials are stored in the container.

### 1.2. Auth Proxy on non-GKE hosts

```bash
cloud-sql-proxy --private-ip --auto-iam-authn $INSTANCE &
psql "host=127.0.0.1 user=iam:$GSA_EMAIL sslmode=disable"
```

This method requires that the host running the Cloud SQL Auth Proxy has direct private IP connectivity to the Cloud SQL instance, meaning it must be within the same VPC network or a network connected via VPC peering or VPN. It will **not work** from local environments unless a secure network connection (e.g., VPN or Cloud Interconnect) to the Google Cloud VPC network is established.

The proxy obtains IAM authentication tokens automatically using the host's Application Default Credentials or the VM's assigned service account. ([Auth Proxy guide](https://cloud.google.com/sql/docs/postgres/connect-auth-proxy))

## 2. FAQs

**Q – Does IAM auth work for read replicas?**
If the `cloudsql.iam_authentication` flag is **enabled on the primary**, Cloud SQL automatically enables it on any read replicas created afterwards. If it's **disabled on the primary**, replicas can't use IAM authentication and the flag cannot be turned on later ([Read‑replica creation guide](https://cloud.google.com/sql/docs/postgres/replication/create-replica)).

**Q – Which editions support IAM auth?**
Both **Enterprise** and **Enterprise Plus** editions support IAM authentication; Cloud SQL defaults to Enterprise Plus for PostgreSQL 16+ and Enterprise for earlier versions ([Edition overview](https://cloud.google.com/sql/docs/postgres/editions-intro)).

## 3. Terraform IAM examples

TODO

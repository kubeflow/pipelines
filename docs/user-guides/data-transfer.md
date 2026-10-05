# Export and import pipeline data

Use **Export / Import** in the Pipelines navigation to move data between
installations with separate databases. Both installations can continue accepting
new work during the overlap. This transfers metadata and definitions; artifact
files stay at their existing storage locations.

## Export from the source

1. Choose your namespace and open **Export / Import**.
2. Optionally select a completed-history UTC time range. The range limits completed
   runs only; every archive includes all experiments, pipeline definitions and
   versions, tags, and schedules in the namespace, including unused definitions
   and empty experiments.
3. Select **Download archive** and save the downloaded JSON archive.

Active executions are excluded. Let old runs finish and export another batch
before retiring the old installation. Keep selected completed runs and their
metadata unchanged while exporting: SQL, Kubernetes resources, and (on 2.18)
MLMD do not share one transaction.

Archives can contain parameters, runtime manifests and storage references. Share
and retain them as you would the original namespace data. An archive contains no
artifact file bytes and no pod log files.

## Validate and import in the destination

1. Open the destination installation, select the same namespace name, and open
   **Export / Import**.
2. Select the archive. A name prefix helps avoid collisions with existing
   experiments and pipelines, such as the destination's `Default` experiment.
3. Select **Validate archive**. Review the resource counts and warnings. Resolve
   any reported conflicts before continuing.
4. Select **Import metadata**. Validation does not reserve the destination indefinitely;
   import rechecks permissions and conflicts before publishing data.

Changing the file, name prefix or namespace invalidates the preview. The server
also validates requests independently of the UI. Identical repeated imports
reuse their own previously imported data; they do not overwrite unrelated native
resources. Use the same archive and options after an interrupted request. This is a
snapshot transfer, not continuous synchronization: changes to previously imported
definitions can conflict with later batches. Adding pipeline versions is supported;
existing explicit Kubernetes-catalog defaults are retained. SQL catalogs keep
their normal newest-version selection, so adding a newer version can change which
version a floating schedule uses. Keep existing version contents and schedule
definitions stable until the transfer is complete.

Imported resource IDs may change, especially when Kubernetes assigns catalog or
schedule IDs. Relationships inside the archive are mapped to the destination IDs.
The destination keeps its own default-experiment setting, migration state and
runtime configuration. Namespace remapping and cross-generation imports are not
supported. Source and destination must use the same database engine and matching
release schemas. Use the transfer implementation from the
same storage generation: master/native archives and release-2.18/MLMD archives
are different formats.

## Schedules and historical runs

Schedules are created as valid destination schedules with **Enabled = false** and
**No catch-up = true**. Source scheduling progress is not copied. Review their
service account, parameters and trigger settings in **Recurring runs** before
explicitly enabling them. Catch-up stays disabled unless you choose to change it,
so enabling an old schedule does not replay its missed intervals automatically.
Single-user installations with runtime plugins require catalog-backed schedules;
recreate inline-only schedules against a saved pipeline definition before export.

Completed runs remain history. They can be viewed, archived and deleted; attempts
to retry, terminate, recreate their imported ID, or report runtime changes are
rejected. Submit a new run to execute the pipeline in the destination. Imported
history is excluded from execution-cache reuse. Destination retention policies
still apply to original finish times and archive states.

## Shared artifact storage

When the installations use the same bucket and paths, there is nothing to copy.
The destination must have permission and configuration to read the original URIs.
Import does not verify every external object or grant bucket access. Keep the
bucket and required credentials after deleting the source cluster.

Archived logs may remain readable if the destination's archive configuration
matches their original locations. Live source pod logs are not transferred.

## Permissions, limits and recovery

Export requires namespace read permissions for experiments, pipelines, runs and
schedules. Import requires namespace list/create permissions and applicable
service-account permissions for imported schedules. Shared-read mode does not
bypass transfer authorization. Native-storage installations also require the
corresponding artifact permissions. Users never provide database credentials.

Archives are limited to 256 MiB of metadata. Use smaller completed-history time ranges
for larger histories; retain the same name prefix across batches. One transfer at
a time runs per API-server process; a busy server asks the client to retry. Native
archives also have limits of 1,000 completed runs, 20,000 history rows, and
10,000 catalog/schedule objects. Narrow time ranges for history limits; the
complete catalog must fit in one archive.

Validation does not create Kubernetes resources or write MLMD. A real import can
stage catalog objects, **disabled** schedule objects and (on 2.18) metadata before
its SQL commit. These systems cannot commit atomically. If interrupted, staged
objects may be visible even though SQL history has not been published. Retry the
same archive and options: provenance and content checks allow reuse of the
transfer's own staged resources. Do not enable or edit staged schedules or remove
shared staged metadata while recovering an import. A lost response during SQL
commit can mean the commit succeeded; repeat import detects the existing receipt.

## HTTP API

These are authenticated HTTP file-transfer endpoints, not gRPC methods. They use
the installation's normal authentication mechanism. The machine-readable contract
is maintained in `backend/api/v2beta1/transfer.openapi.yaml`.

- `POST /apis/v2beta1/transfer/export?namespace=NAME`, with
  `Content-Type: application/json` and body `{}`. Optional `completed_after` and
  `completed_before` values are Unix timestamps in seconds.
- `POST /apis/v2beta1/transfer/import?namespace=NAME&dry_run=true&name_prefix=imported-`,
  with the raw archive as the `application/json` request body. Validation is the
  default; set `dry_run=false` to apply.

Import responses contain resource `counts`, `imported` and `skipped` counts,
`dry_run`, and `warnings`. Upload and download archive bytes without parsing and
reserializing numbers in JavaScript; doing so can lose precision above `2^53`.

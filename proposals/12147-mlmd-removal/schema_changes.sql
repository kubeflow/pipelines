CREATE TABLE `artifacts`
(
    `UUID`            varchar(191) NOT NULL,
    `Namespace`       varchar(63)  NOT NULL,              -- enables multi-tenancy on artifacts
    `Type`            varchar(64)           DEFAULT NULL, -- examples: Artifact, Model, Dataset
    -- URI is immutable, reject in API server call if update artifact attempts to change the URI for a pre-existing artifact
    `Uri`             text,
    `Name`            varchar(128)          DEFAULT NULL,
    `Description`     varchar(128)          DEFAULT NULL,
    `CreatedAtInSec`  bigint       NOT NULL DEFAULT '0',
    `LastUpdateInSec` bigint       NOT NULL DEFAULT '0',
    `Metadata`        JSON                  DEFAULT NULL, -- equivalent to mlmd custom properties
    `NumberValue`     bigint                DEFAULT NULL, -- used for metrics

    PRIMARY KEY (`UUID`),
    KEY               idx_type_namespace (`Namespace`, `Type`),
    KEY               idx_created_timestamp (`CreatedAtInSec`),
    KEY               idx_last_update_timestamp (`LastUpdateInSec`)
);

-- Analogous to an mlmd Event, except it is specific to artifacts <-> tasks (instead of executions)
CREATE TABLE `artifact_tasks`
(
    `UUID`              varchar(191) NOT NULL,
    `ArtifactID`        varchar(191) NOT NULL,
    `TaskID`            varchar(191) NOT NULL,
    -- 0 for INPUT, 1 for OUTPUT
    `Type`              int          NOT NULL,

    `RunUUID`           varchar(191) NOT NULL,
    `ProducerTaskName`  varchar(128) NOT NULL,
    `ProducerKey`       varchar(128) NOT NULL,
    `ArtifactKey`       varchar(128) NOT NULL,

    PRIMARY KEY (`UUID`),
    UNIQUE KEY `UniqueLink` (`ArtifactID`,`TaskID`,`Type`),
    KEY              `idx_link_task_id` (`TaskID`),
    KEY              `idx_link_artifact_id` (`ArtifactID`),
    KEY              `idx_created_timestamp` (`CreatedAtInSec`),

    KEY `idx_run_uuid` (`RunUUID`),
    CONSTRAINT `fk_artifact_tasks_run_details` FOREIGN KEY (`RunUUID`) REFERENCES `run_details` (`UUID`) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT fk_artifact_tasks_tasks FOREIGN KEY (TaskID) REFERENCES tasks (UUID) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT fk_artifact_tasks_artifacts FOREIGN KEY (ArtifactID) REFERENCES artifacts (UUID) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE `tasks`
(
    `UUID`             varchar(191) NOT NULL,
    `Namespace`        varchar(63)  NOT NULL, -- updated to 63 (max namespace size in k8s)
    -- This is used for searching for cached_fingerprints today
    -- likely to prevent caching across pipelines 
    `PipelineName`        varchar(128) NOT NULL,
    `RunUUID`             varchar(191) NOT NULL,
    `Pods`                json         NOT NULL, -- This is broken today and will need to be fixed
    `CreatedAtInSec`      bigint       NOT NULL,
    `StartedInSec`        bigint       DEFAULT '0',
    `FinishedInSec`       bigint       DEFAULT '0',
    `Fingerprint`         varchar(255) NOT NULL,
    `Name`                varchar(128) DEFAULT NULL,
    `DisplayName`         varchar(128) DEFAULT NULL,
    `ParentTaskUUID`      varchar(191) DEFAULT NULL,
    `Status`              varchar(64)  DEFAULT NULL,
    `StatusMetadata`      json         DEFAULT NULL,
    `InputParameters`     json,
    `OutputParameters`    json,
    -- Corresponds to the executions created for each driver pod, which result in a Node on the Run Graph.
    -- E.g values are: Runtime, Condition, Loop, etc.
    `Type`             varchar(64)  NOT NULL,
    -- All type-specific attributes (Runtime.DisplayName, Loop.IterationIndex/Count)
    `TypeAttrs`        json         NOT NULL,

    PRIMARY KEY (`UUID`),
    KEY                idx_task_type (`Type`),
    KEY                idx_pipeline_name (`PipelineName`),
    KEY                idx_parent_run (`RunUUID`, `ParentTaskUUID`),
    KEY                idx_parent_task_uuid (`ParentTaskUUID`),
    KEY                idx_created_timestamp (`CreatedAtInSec`),
    KEY                idx_started_timestamp (`StartedInSec`),
    KEY                idx_finished_timestamp (`FinishedInSec`),
    CONSTRAINT `fk_tasks_parent_task` FOREIGN KEY (`ParentTaskUUID`) REFERENCES tasks (`UUID`) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT `tasks_RunUUID_run_details_UUID_foreign` FOREIGN KEY (`RunUUID`) REFERENCES `run_details` (`UUID`) ON DELETE CASCADE ON UPDATE CASCADE
);

# Kubeflow Pipelines ML-Metadata (MLMD) Removal Proposal

This proposal outlines a comprehensive plan to remove ML-Metadata (MLMD) from Kubeflow Pipelines (KFP) v2 and replace its functionality with native KFP API server capabilities. The goal is to simplify the architecture, reduce dependencies, improve maintainability, and provide better integration with existing KFP infrastructure.

## Motivation

KFP v2 currently uses ML-Metadata (MLMD) to track pipeline executions, store artifacts and metadata, manage execution parameters and metrics, and provide lineage tracking and caching capabilities.

However, MLMD introduces significant operational and architectural challenges:

**Operational Complexity:**
- Additional services to deploy and maintain
- Complex KFP-MLMD integration points
- Difficult debugging and troubleshooting
- Lack of control over a key component of KFP

**Technical Limitations:**
- Separate database schema misaligned with KFP's native structure
- Limited control over metadata storage and querying
- Lack of multi-tenancy support for artifacts
- Blocks MySQL upgrades beyond 8.x and PostgreSQL support
- MLMD project is in maintenance mode and no longer actively maintained, leading to stagnated development and bug fixes

Removing MLMD and implementing native metadata management will eliminate these pain points while maintaining full functionality, resulting in a simpler, more maintainable system.

## User Stories

- As a Data Scientist, I want to track my pipeline executions and their states without relying on external services, so I can have a simpler and more reliable system.
- As an ML Engineer, I want to store and retrieve input/output artifacts and their metadata directly through KFP API, so I can have better control and visibility over my pipeline data.
- As a KFP Admin, I want to reduce the number of services I need to maintain, so I can improve system reliability and reduce operational overhead.
- As a KFP Admin, I want to ensure that artifact endpoints are properly secured and authenticated, so I can maintain data privacy and prevent unauthorized access.
- As a Developer, I want to have a simplified architecture with fewer integration points, so I can more easily debug and troubleshoot pipeline issues.

## Risks and Mitigations

1. Data Migration Risk
    - Risk: Loss or corruption of existing MLMD data during migration
    - Mitigation: Implement robust data migration tools with validation, and backups before migration
2. Feature Parity
    - Risk: Missing or incomplete implementation of current MLMD functionality
    - Mitigation: Comprehensive feature audit, thorough testing plan, and regression testing 

## Design Details 

The design details can be found [here](design-details.md).

## Delivery Plan

* Add the proto files, tables, and gorm model changes
* Add API Server Logic
* Start adding all _post_ server logic in Driver and Launcher (alongside MLMD)
* Update the UI to start reading from the API server, removing MLMD logic from frontend
* Update resolve/input logic to use Task details and remove all MLMD invocations from the backend
* Remove MLMD writer and associated CI
* Remove MLMD logic from Driver/Launcher, and add Migration logic
* Remove MLMD from manifests

## Conclusion

Removing MLMD from KFP will significantly simplify the architecture while maintaining full functionality. The proposed approach provides a clear migration path with minimal risk and substantial long-term benefits. The phased implementation ensures careful validation at each step while allowing for early feedback and course correction.

This migration will result in a more maintainable, performant, and operationally simple KFP deployment that better serves the needs of ML practitioners and platform operators.

# Data Model: Prototype-Driven Development Pipelines

**Feature**: 017-prototype-driven-development
**Date**: 2026-02-02

## Overview

This document defines the data structures and artifacts used by the prototype pipeline. These models map to JSON schemas for contract validation and YAML structures for pipeline configuration.

---

## Core Entities

### 1. Specification

The primary output of the spec phase, capturing all requirements for the feature.

```
Specification
├── title: string (required, min 5 chars)
├── description: string (required, min 50 chars)
├── user_stories: UserStory[] (required, min 1)
├── entities: Entity[] (required)
├── interfaces: Interface[] (optional)
├── edge_cases: string[] (optional)
└── success_metrics: string[] (optional)
```

**Fields**:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| title | string | Yes | Feature name (5+ characters) |
| description | string | Yes | Feature overview (50+ characters) |
| user_stories | UserStory[] | Yes | At least one user story |
| entities | Entity[] | Yes | Data model entities |
| interfaces | Interface[] | No | API/CLI/UI interfaces |
| edge_cases | string[] | No | Known edge cases |
| success_metrics | string[] | No | Measurable outcomes |

**Validation Rules**:
- `title` must be unique within the project
- `description` must explain the "why" not just the "what"
- At least one user story with acceptance criteria

---

### 2. UserStory

Captures a single user requirement with acceptance criteria.

```
UserStory
├── id: string (required, pattern: "US-[0-9]+")
├── as_a: string (required)
├── i_want: string (required)
├── so_that: string (required)
├── acceptance_criteria: string[] (required, min 1)
└── priority: enum["P1", "P2", "P3", "P4"] (optional)
```

**Fields**:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | string | Yes | Unique ID (US-001, US-002, etc.) |
| as_a | string | Yes | User role/persona |
| i_want | string | Yes | Desired action |
| so_that | string | Yes | Business value |
| acceptance_criteria | string[] | Yes | Testable conditions |
| priority | enum | No | P1 (critical) to P4 (nice-to-have) |

**Validation Rules**:
- `id` must be unique within the specification
- `priority` defaults to P2 if not specified

---

### 3. Entity

Represents a data model entity with fields and relationships.

```
Entity
├── name: string (required)
├── fields: Field[] (required)
├── relationships: string[] (optional)
└── validation_rules: string[] (optional)
```

**Fields**:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| name | string | Yes | Entity name (PascalCase) |
| fields | Field[] | Yes | Entity fields |
| relationships | string[] | No | Related entity names |
| validation_rules | string[] | No | Business rules |

**Field Sub-structure**:

```
Field
├── name: string (required)
├── type: string (required)
├── required: boolean (optional, default: false)
├── default: any (optional)
└── constraints: string[] (optional)
```

---

### 4. Interface

Describes an API, CLI, or UI interface.

```
Interface
├── type: enum["cli", "api", "ui", "library"] (required)
├── name: string (required)
├── description: string (optional)
└── operations: Operation[] (optional)
```

**Operation Sub-structure**:

```
Operation
├── name: string (required)
├── description: string (optional)
├── inputs: Parameter[] (optional)
├── outputs: Parameter[] (optional)
└── errors: string[] (optional)
```

---

### 5. DocumentationManifest

Output of the docs phase, tracking generated documentation.

```
DocumentationManifest
├── generated_files: GeneratedFile[] (required, min 1)
└── spec_coverage: SpecCoverage (required)
```

**GeneratedFile**:

```
GeneratedFile
├── path: string (required)
├── type: enum["overview", "user_guide", "api_reference", "architecture"] (required)
└── description: string (required)
```

**SpecCoverage**:

```
SpecCoverage
├── user_stories_documented: string[] (required)
└── entities_documented: string[] (required)
```

---

### 6. DummyManifest

Output of the dummy phase, describing the prototype.

```
DummyManifest
├── prototype_path: string (required)
├── interfaces_implemented: ImplementedInterface[] (required)
├── runnable: boolean (required)
├── entry_point: string (optional, required if runnable=true)
└── todo_markers: TodoMarker[] (optional)
```

**ImplementedInterface**:

```
ImplementedInterface
├── name: string (required)
├── stub_type: enum["hardcoded", "mock", "noop", "echo"] (required)
└── file_path: string (optional)
```

**TodoMarker**:

```
TodoMarker
├── file: string (required)
├── line: integer (required)
└── description: string (required)
```

---

## Artifact Flow Diagram

```
┌──────────────────────────────────────────────────────────────────────────┐
│                              SPEC PHASE                                   │
│                                                                          │
│  Input: user_description (string)                                        │
│                                                                          │
│  ┌─────────────────┐         ┌─────────────────────────┐                │
│  │  spec-navigate  │────────▶│      spec-define        │                │
│  │                 │         │                         │                │
│  │  Output:        │         │  Output:                │                │
│  │  codebase_      │         │  specification.json     │                │
│  │  context.json   │         │  (Specification)        │                │
│  └─────────────────┘         └────────────┬────────────┘                │
│                                           │                              │
└───────────────────────────────────────────┼──────────────────────────────┘
                                            │
                                            ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                              DOCS PHASE                                   │
│                                                                          │
│  Input: specification.json                                               │
│                                                                          │
│  ┌─────────────────────────┐                                            │
│  │     docs-generate       │                                            │
│  │                         │                                            │
│  │  Output:                │                                            │
│  │  - feature-docs.md      │                                            │
│  │  - api-docs.md          │                                            │
│  │  - docs-manifest.json   │                                            │
│  │    (DocumentationManifest) │                                         │
│  └────────────┬────────────┘                                            │
│               │                                                          │
└───────────────┼──────────────────────────────────────────────────────────┘
                │
                ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                              DUMMY PHASE                                  │
│                                                                          │
│  Input: specification.json, feature-docs.md                              │
│                                                                          │
│  ┌─────────────────┐         ┌─────────────────────────┐                │
│  │  dummy-scaffold │────────▶│     dummy-verify        │                │
│  │                 │         │                         │                │
│  │  Output:        │         │  Output:                │                │
│  │  - prototype/   │         │  dummy-verification.md  │                │
│  │  - interfaces.  │         │                         │                │
│  │    json         │         │                         │                │
│  │  - dummy-       │         │                         │                │
│  │    manifest.json│         │                         │                │
│  │  (DummyManifest)│         │                         │                │
│  └─────────────────┘         └────────────┬────────────┘                │
│                                           │                              │
└───────────────────────────────────────────┼──────────────────────────────┘
                                            │
                                            ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                           IMPLEMENT PHASE                                 │
│                                                                          │
│  Input: All prior artifacts                                              │
│                                                                          │
│  ┌──────────────┐   ┌──────────────┐   ┌──────────────────┐            │
│  │implement-plan│──▶│implement-code│──▶│ implement-review │            │
│  │              │   │              │   │                  │            │
│  │Output:       │   │Output:       │   │Output:           │            │
│  │implementation│   │source/       │   │final-review.md   │            │
│  │-plan.md      │   │tests/        │   │                  │            │
│  └──────────────┘   └──────────────┘   └──────────────────┘            │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

---

## State Tracking

### StepState (Existing Wave Entity)

Used to track phase completion and support resume functionality.

```
StepState
├── pipeline_run_id: string
├── step_id: string
├── state: enum["pending", "running", "completed", "failed", "retrying"]
├── started_at: timestamp
├── completed_at: timestamp (nullable)
├── retry_count: integer
└── error_message: string (nullable)
```

### ArtifactRecord (Existing Wave Entity)

Used to track artifact timestamps for stale detection.

```
ArtifactRecord
├── pipeline_run_id: string
├── step_id: string
├── name: string
├── path: string
├── type: string
├── created_at: timestamp
└── checksum: string (optional)
```

---

## Relationships

```
Pipeline Run (1) ──────── (*) Step State
     │
     └──────────────────── (*) Artifact Record

Specification (1) ─────── (*) User Story
     │
     ├──────────────────── (*) Entity
     │
     └──────────────────── (*) Interface

Documentation Manifest (1) ─── (*) Generated File
     │
     └────────────────────────── (1) Spec Coverage

Dummy Manifest (1) ───────── (*) Implemented Interface
     │
     └────────────────────── (*) Todo Marker
```

---

## JSON Schema Files

The following JSON schema files will be created in `.wave/contracts/`:

1. **spec-phase.schema.json** - Validates Specification output
2. **docs-phase.schema.json** - Validates DocumentationManifest output
3. **dummy-phase.schema.json** - Validates DummyManifest output

See the [plan.md](./plan.md) Contract Definitions section for complete schema definitions.

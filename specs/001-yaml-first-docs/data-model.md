# Data Model: Documentation Content Types

**Feature**: 001-yaml-first-docs
**Date**: 2026-02-03
**Purpose**: Define the content entity relationships and validation rules

## Content Entities

### Workflow YAML
**Description**: Primary interface for defining AI workflows, the core entity that everything else supports.

**Attributes**:
- apiVersion (string): Version compatibility
- kind (string): Always "Pipeline"
- metadata (object): Name, description, version
- spec (object): Steps, personas, contracts, deliverables
- dependencies (array): Required artifacts from other steps

**Relationships**:
- Contains multiple WorkflowSteps
- References PersonaConfigurations
- Produces Deliverables
- Validated by Contracts

**Validation Rules**:
- Must follow valid YAML schema
- All persona references must exist
- Step dependencies must form valid DAG
- Contract schemas must be syntactically valid

---

### Deliverable
**Description**: Any output artifact produced by a workflow step.

**Attributes**:
- type (string): file, report, commit, pr, message
- path (string): Output location or identifier
- format (string): json, markdown, typescript, etc.
- validation_contract (reference): Associated contract

**Relationships**:
- Produced by WorkflowStep
- Validated by Contract
- Can be input to subsequent WorkflowStep

**Validation Rules**:
- Type must be supported deliverable type
- Path must be valid for deliverable type
- Format must match contract expectations

---

### Contract
**Description**: Validation schema that guarantees deliverable quality.

**Attributes**:
- type (string): json_schema, typescript, test_suite
- schema_path (string): Path to validation definition
- enforcement_level (string): strict, warning, disabled

**Relationships**:
- Validates Deliverable
- Referenced by WorkflowStep
- May reference external schema files

**Validation Rules**:
- Schema path must exist and be valid
- Type must be supported contract type
- Enforcement level must be valid option

---

### Community Workflow
**Description**: Shareable, versioned workflow templates from the ecosystem.

**Attributes**:
- name (string): Unique identifier
- version (string): Semantic version
- author (string): Creator/maintainer
- description (string): Purpose and usage
- category (string): security, testing, docs, etc.
- source_url (string): Repository location

**Relationships**:
- Extends WorkflowYAML
- Can be imported into user workflows
- May depend on other CommunityWorkflows

**Validation Rules**:
- Name must be unique within registry
- Version must follow semver
- Source URL must be accessible
- Base workflow must be valid

## Content Hierarchy

```
ParadigmConcept
├── InfrastructureParallel (1:many)
├── WorkflowExample (1:many)
└── ValueProposition (1:1)

WorkflowDocumentation
├── WorkflowYAML (1:1)
├── UsageExample (1:many)
├── PersonaConfiguration (1:many)
└── ContractDefinition (1:many)

ConceptualExplanation
├── PersonaDetail (1:many)
├── ArchitecturalPrinciple (1:many)
└── TechnicalImplementation (1:many)

ReferenceDocumentation
├── SchemaDefinition (1:many)
├── CLICommand (1:many)
└── TroubleshootingGuide (1:many)
```

## State Transitions

### User Journey States
- **Discovery**: User encounters paradigm concept
- **Understanding**: User grasps Infrastructure-as-Code parallel
- **Experimentation**: User creates first workflow YAML
- **Adoption**: User integrates workflows into development process
- **Mastery**: User shares workflows and adopts community patterns

### Content Validation States
- **Draft**: Content exists but not validated
- **Reviewed**: Content passes technical accuracy check
- **User-Tested**: Content validated with target user persona
- **Published**: Content available in documentation site
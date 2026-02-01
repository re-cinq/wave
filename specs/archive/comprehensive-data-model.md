# Data Model: Wave - Comprehensive Claude Code Integration System

**Plan**: [comprehensive-implementation-plan.md](comprehensive-implementation-plan.md)  
**Created**: 2025-02-01  
**Version**: 1.0

## Core Data Entities

### System Configuration

```yaml
# Wave System Configuration
version: "1.0"
server:
  host: "localhost"
  port: 8080
  daemon_mode: false
  log_level: "info"

security:
  authentication:
    type: "file" # file, ldap, oauth
    config_file: "configs/auth.yaml"
  
  personas:
    config_dir: "configs/personas"
    default_persona: "developer"
    enforce_boundaries: true

pipeline:
  max_concurrent: 100
  default_timeout: "30m"
  workspace_cleanup: "1h"

context:
  max_size: "100MB"
  compaction_threshold: "80%"
  persistence_enabled: true
  storage_dir: "/tmp/wave/contexts"

workspace:
  base_dir: "/tmp/wave/workspaces"
  isolation_type: "namespace" # namespace, chroot, simple
  cleanup_delay: "5m"

claude:
  binary_path: "/usr/local/bin/claude-code"
  timeout: "10m"
  max_retries: 3
  workspace_dir: "/tmp/wave/claude"
```

### Persona Definition

```yaml
# Persona Configuration Structure
persona:
  name: "developer"
  display_name: "Developer"
  description: "General development tasks and code modifications"
  
  security_context:
    user_id: "${USER_ID}"
    group_id: "${GROUP_ID}"
    permissions:
      - "code:read"
      - "code:write"
      - "pipeline:execute"
      - "workspace:create"
    
    constraints:
      - type: "directory_access"
        allowed_paths:
          - "/home/${USER}/projects"
          - "/tmp/wave/workspaces/${SESSION_ID}"
        denied_paths:
          - "/etc"
          - "/usr/local/bin"
      
      - type: "resource_limits"
        max_memory: "2GB"
        max_cpu: "50%"
        max_processes: 10
      
      - type: "time_constraints"
        max_execution_time: "2h"
        allowed_hours: "9-17"  # Business hours

  capabilities:
    - "code_analysis"
    - "file_modification"
    - "process_execution"
    - "pipeline_creation"

  tools_allowed:
    - "git"
    - "go"
    - "npm"
    - "docker"
    - "texteditors"

  ai_context:
    system_prompt: |
      You are a Developer persona with access to code modification tools.
      Focus on implementing features, fixing bugs, and improving code quality.
      Follow established coding standards and security practices.
    
    temperature: 0.7
    max_tokens: 4000
```

### Pipeline Definition

```yaml
# Pipeline Execution Structure
pipeline:
  id: "feature-development"
  name: "Feature Development Pipeline"
  version: "1.0"
  description: "Standard pipeline for feature development tasks"
  
  metadata:
    created_by: "system"
    created_at: "2025-02-01T00:00:00Z"
    tags: ["development", "feature"]
  
  persona: "developer"
  workspace: "ephemeral"
  
  tasks:
    - id: "analyze-requirements"
      name: "Analyze Requirements"
      type: "claude-code"
      persona: "analyst"
      
      inputs:
        - name: "requirements"
          type: "text"
          required: true
      
      prompt_template: |
        Analyze the following requirements and provide a detailed implementation plan:
        
        Requirements: {{requirements}}
        
        Please provide:
        1. Technical approach
        2. Required components
        3. Implementation steps
        4. Testing strategy
        5. Risk assessment
      
      outputs:
        - name: "implementation_plan"
          type: "json"
          schema: "implementation-plan.json"
      
      timeout: "15m"
      retry_count: 2
    
    - id: "implement-feature"
      name: "Implement Feature"
      type: "claude-code"
      persona: "developer"
      
      dependencies: ["analyze-requirements"]
      
      inputs:
        - name: "implementation_plan"
          type: "json"
          from_task: "analyze-requirements"
          output_name: "implementation_plan"
      
      prompt_template: |
        Implement the following feature according to the implementation plan:
        
        Plan: {{implementation_plan}}
        
        Please:
        1. Write the code following best practices
        2. Add appropriate tests
        3. Update documentation
        4. Ensure code quality standards
      
      workspace_mounts:
        - source: "./src"
          destination: "/workspace/src"
        - source: "./tests"
          destination: "/workspace/tests"
      
      tools_allowed:
        - "go"
        - "git"
        - "texteditors"
      
      timeout: "60m"
      retry_count: 1
    
    - id: "review-code"
      name: "Code Review"
      type: "claude-code"
      persona: "reviewer"
      
      dependencies: ["implement-feature"]
      
      inputs:
        - name: "changed_files"
          type: "file_list"
          from_task: "implement-feature"
          output_name: "changed_files"
      
      prompt_template: |
        Review the following code changes:
        
        Changed files: {{changed_files}}
        
        Please review for:
        1. Code quality and standards
        2. Security vulnerabilities
        3. Performance considerations
        4. Test coverage
        5. Documentation completeness
      
      outputs:
        - name: "review_report"
          type: "json"
          schema: "code-review.json"
      
      timeout: "30m"
      retry_count: 1

  execution:
    parallel: true
    max_parallel_tasks: 3
    on_failure: "continue" # continue, stop, rollback
  
  notifications:
    on_start:
      - type: "webhook"
        url: "${WEBHOOK_URL}"
    on_complete:
      - type: "email"
        recipients: ["${USER_EMAIL}"]
    on_failure:
      - type: "slack"
        channel: "#dev-alerts"
```

### Context Management

```go
// Context Data Structures
type Context struct {
    ID          string                 `json:"id"`
    Type        ContextType           `json:"type"`
    Persona     string                `json:"persona"`
    Pipeline    string                `json:"pipeline"`
    Task        string                `json:"task"`
    
    CreatedAt   time.Time             `json:"created_at"`
    UpdatedAt   time.Time             `json:"updated_at"`
    ExpiresAt   time.Time             `json:"expires_at"`
    
    Data        map[string]interface{} `json:"data"`
    Metadata    ContextMetadata       `json:"metadata"`
    
    Size        int64                 `json:"size"`
    TokenCount  int                   `json:"token_count"`
    
    Compression CompressionType      `json:"compression"`
    Persistence PersistenceType       `json:"persistence"`
}

type ContextType string
const (
    ContextTypeTask      ContextType = "task"
    ContextTypePipeline  ContextType = "pipeline"
    ContextTypeSession   ContextType = "session"
    ContextTypeSystem    ContextType = "system"
)

type ContextMetadata struct {
    Tags        []string          `json:"tags"`
    Owner       string            `json:"owner"`
    Scope       string            `json:"scope"`
    Priority    int               `json:"priority"`
    Version     string            `json:"version"`
    Annotations map[string]string `json:"annotations"`
}

type CompressionType string
const (
    CompressionNone   CompressionType = "none"
    CompressionGzip   CompressionType = "gzip"
    CompressionLZ4    CompressionType = "lz4"
)

type PersistenceType string
const (
    PersistenceMemory PersistenceType = "memory"
    PersistenceFile   PersistenceType = "file"
    PersistenceDB     PersistenceType = "database"
)
```

### Workspace Management

```go
// Workspace Data Structures
type Workspace struct {
    ID          string           `json:"id"`
    Type        WorkspaceType    `json:"type"`
    State       WorkspaceState   `json:"state"`
    
    CreatedAt   time.Time        `json:"created_at"`
    UpdatedAt   time.Time        `json:"updated_at"`
    ExpiresAt   time.Time        `json:"expires_at"`
    
    Path        string           `json:"path"`
    Size        int64            `json:"size"`
    
    Owner       string           `json:"owner"`
    Persona     string           `json:"persona"`
    Pipeline    string           `json:"pipeline"`
    Task        string           `json:"task"`
    
    Resources   ResourceLimits   `json:"resources"`
    Mounts      []MountPoint     `json:"mounts"`
    Processes   []ProcessInfo    `json:"processes"`
}

type WorkspaceType string
const (
    WorkspaceTypeEphemeral WorkspaceType = "ephemeral"
    WorkspaceTypePersistent WorkspaceType = "persistent"
    WorkspaceTypeShared    WorkspaceType = "shared"
)

type WorkspaceState string
const (
    WorkspaceStateCreating    WorkspaceState = "creating"
    WorkspaceStateActive     WorkspaceState = "active"
    WorkspaceStateSuspended  WorkspaceState = "suspended"
    WorkspaceStateCleaning   WorkspaceState = "cleaning"
    WorkspaceStateDeleted    WorkspaceState = "deleted"
)

type ResourceLimits struct {
    MaxMemory    int64 `json:"max_memory"`    // bytes
    MaxCPU       int   `json:"max_cpu"`       // percentage
    MaxProcesses int   `json:"max_processes"`
    MaxDisk      int64 `json:"max_disk"`      // bytes
    MaxFiles     int   `json:"max_files"`
}

type MountPoint struct {
    Source      string            `json:"source"`
    Destination string            `json:"destination"`
    Type        MountType         `json:"type"`
    Options     map[string]string `json:"options"`
    ReadOnly    bool              `json:"read_only"`
}

type MountType string
const (
    MountTypeBind   MountType = "bind"
    MountTypeTmpfs  MountType = "tmpfs"
    MountTypeVolume MountType = "volume"
)
```

### Message Communication

```go
// Relay System Data Structures
type Message struct {
    ID          string        `json:"id"`
    Type        MessageType   `json:"type"`
    Version     string        `json:"version"`
    
    From        string        `json:"from"`
    To          string        `json:"to"`
    ReplyTo     string        `json:"reply_to,omitempty"`
    
    Timestamp   time.Time     `json:"timestamp"`
    TTL         time.Duration `json:"ttl,omitempty"`
    
    Payload     interface{}   `json:"payload"`
    Headers     MessageHeaders `json:"headers"`
    
    Priority    int           `json:"priority"`
    Correlation string        `json:"correlation_id,omitempty"`
}

type MessageType string
const (
    MessageTypeRequest    MessageType = "request"
    MessageTypeResponse   MessageType = "response"
    MessageTypeEvent      MessageType = "event"
    MessageTypeCommand    MessageType = "command"
    MessageTypeNotification MessageType = "notification"
)

type MessageHeaders struct {
    ContentType string            `json:"content_type"`
    Encoding    string            `json:"encoding"`
    Tags        []string          `json:"tags"`
    Metadata    map[string]string `json:"metadata"`
}

// Specific Message Types
type ClaudeRequest struct {
    Prompt       string            `json:"prompt"`
    Context      map[string]interface{} `json:"context"`
    Persona      string            `json:"persona"`
    Workspace    string            `json:"workspace"`
    Tools        []string          `json:"tools"`
    Constraints  map[string]interface{} `json:"constraints"`
}

type ClaudeResponse struct {
    Content      string            `json:"content"`
    Actions      []Action          `json:"actions"`
    TokensUsed   int               `json:"tokens_used"`
    Duration     time.Duration     `json:"duration"`
    Success      bool              `json:"success"`
    Error        string            `json:"error,omitempty"`
    Context      map[string]interface{} `json:"context"`
}

type Action struct {
    Type        ActionType         `json:"type"`
    Target      string             `json:"target"`
    Parameters  map[string]interface{} `json:"parameters"`
    Required    bool               `json:"required"`
    Validate    ValidationRule     `json:"validate,omitempty"`
}

type ActionType string
const (
    ActionTypeFileWrite   ActionType = "file_write"
    ActionTypeFileRead    ActionType = "file_read"
    ActionTypeExecute     ActionType = "execute"
    ActionTypeGitCommand  ActionType = "git_command"
    ActionTypeAPICall     ActionType = "api_call"
)
```

### Validation System

```go
// Validation Data Structures
type ValidationRule struct {
    ID          string             `json:"id"`
    Name        string             `json:"name"`
    Type        ValidationType     `json:"type"`
    Severity    ValidationSeverity `json:"severity"`
    
    Description string             `json:"description"`
    Category    string             `json:"category"`
    
    Condition   string             `json:"condition"`     // Expression to evaluate
    Parameters  map[string]interface{} `json:"parameters"`
    
    Enabled     bool               `json:"enabled"`
    Scope       string             `json:"scope"`        // global, persona, pipeline, task
}

type ValidationType string
const (
    ValidationTypeSecurity    ValidationType = "security"
    ValidationTypePerformance ValidationType = "performance"
    ValidationTypeCompliance  ValidationType = "compliance"
    ValidationTypeQuality     ValidationType = "quality"
    ValidationTypeResource    ValidationType = "resource"
)

type ValidationSeverity string
const (
    SeverityInfo     ValidationSeverity = "info"
    SeverityWarning  ValidationSeverity = "warning"
    SeverityError    ValidationSeverity = "error"
    SeverityCritical ValidationSeverity = "critical"
)

type ValidationResult struct {
    RuleID      string             `json:"rule_id"`
    Passed      bool               `json:"passed"`
    Message     string             `json:"message"`
    Details     map[string]interface{} `json:"details"`
    Severity    ValidationSeverity `json:"severity"`
    Timestamp   time.Time          `json:"timestamp"`
    Context     map[string]interface{} `json:"context"`
}

type ValidationReport struct {
    ID          string             `json:"id"`
    Target      string             `json:"target"`       // What was validated
    Type        ValidationType     `json:"type"`         // Type of validation
    Status      ValidationStatus   `json:"status"`
    
    Results     []ValidationResult `json:"results"`
    Summary     ValidationSummary  `json:"summary"`
    
    Timestamp   time.Time          `json:"timestamp"`
    Duration    time.Duration      `json:"duration"`
}

type ValidationStatus string
const (
    ValidationStatusPassed   ValidationStatus = "passed"
    ValidationStatusFailed   ValidationStatus = "failed"
    ValidationStatusWarning ValidationStatus = "warning"
    ValidationStatusError   ValidationStatus = "error"
)

type ValidationSummary struct {
    Total     int `json:"total"`
    Passed    int `json:"passed"`
    Failed    int `json:"failed"`
    Warnings  int `json:"warnings"`
    Errors    int `json:"errors"`
    Critical  int `json:"critical"`
}
```

### Error Handling

```go
// Error Data Structures
type Error struct {
    Code        string        `json:"code"`
    Type        ErrorType     `json:"type"`
    Severity    ErrorSeverity `json:"severity"`
    
    Message     string        `json:"message"`
    Description string        `json:"description"`
    
    Context     ErrorContext  `json:"context"`
    Cause       error         `json:"-"`           // Go error chain
    Timestamp   time.Time     `json:"timestamp"`
    
    Retryable   bool          `json:"retryable"`
    Suggestions []string      `json:"suggestions"`
}

type ErrorType string
const (
    ErrorTypeValidation    ErrorType = "validation"
    ErrorTypeSecurity      ErrorType = "security"
    ErrorTypeInfrastructure ErrorType = "infrastructure"
    ErrorTypeIntegration   ErrorType = "integration"
    ErrorTypePerformance   ErrorType = "performance"
    ErrorTypeUser          ErrorType = "user"
    ErrorTypeSystem        ErrorType = "system"
)

type ErrorSeverity string
const (
    ErrorSeverityLow      ErrorSeverity = "low"
    ErrorSeverityMedium   ErrorSeverity = "medium"
    ErrorSeverityHigh     ErrorSeverity = "high"
    ErrorSeverityCritical ErrorSeverity = "critical"
)

type ErrorContext struct {
    Component   string            `json:"component"`
    Operation   string            `json:"operation"`
    UserID      string            `json:"user_id,omitempty"`
    SessionID   string            `json:"session_id,omitempty"`
    RequestID   string            `json:"request_id,omitempty"`
    Persona     string            `json:"persona,omitempty"`
    Pipeline    string            `json:"pipeline,omitempty"`
    Task        string            `json:"task,omitempty"`
    Metadata    map[string]interface{} `json:"metadata"`
}
```

## JSON Schema Definitions

### Configuration Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "Wave System Configuration",
  "type": "object",
  "properties": {
    "version": {
      "type": "string",
      "pattern": "^\\d+\\.\\d+$"
    },
    "server": {
      "type": "object",
      "properties": {
        "host": {"type": "string"},
        "port": {"type": "integer", "minimum": 1, "maximum": 65535},
        "daemon_mode": {"type": "boolean"},
        "log_level": {
          "type": "string",
          "enum": ["debug", "info", "warn", "error"]
        }
      },
      "required": ["host", "port"]
    },
    "security": {
      "type": "object",
      "properties": {
        "authentication": {
          "type": "object",
          "properties": {
            "type": {"type": "string", "enum": ["file", "ldap", "oauth"]},
            "config_file": {"type": "string"}
          },
          "required": ["type"]
        },
        "personas": {
          "type": "object",
          "properties": {
            "config_dir": {"type": "string"},
            "default_persona": {"type": "string"},
            "enforce_boundaries": {"type": "boolean"}
          },
          "required": ["config_dir", "default_persona"]
        }
      },
      "required": ["authentication", "personas"]
    }
  },
  "required": ["version", "server", "security"]
}
```

### Pipeline Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "Pipeline Definition",
  "type": "object",
  "properties": {
    "id": {
      "type": "string",
      "pattern": "^[a-z0-9-]+$"
    },
    "name": {"type": "string"},
    "version": {
      "type": "string",
      "pattern": "^\\d+\\.\\d+$"
    },
    "description": {"type": "string"},
    "persona": {"type": "string"},
    "workspace": {
      "type": "string",
      "enum": ["ephemeral", "persistent", "shared"]
    },
    "tasks": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "id": {"type": "string"},
          "name": {"type": "string"},
          "type": {"type": "string"},
          "persona": {"type": "string"},
          "dependencies": {
            "type": "array",
            "items": {"type": "string"}
          },
          "inputs": {
            "type": "array",
            "items": {
              "type": "object",
              "properties": {
                "name": {"type": "string"},
                "type": {"type": "string"},
                "required": {"type": "boolean"}
              },
              "required": ["name", "type"]
            }
          },
          "prompt_template": {"type": "string"},
          "timeout": {
            "type": "string",
            "pattern": "^\\d+[smh]$"
          },
          "retry_count": {
            "type": "integer",
            "minimum": 0,
            "maximum": 5
          }
        },
        "required": ["id", "name", "type", "prompt_template"]
      }
    }
  },
  "required": ["id", "name", "version", "tasks"]
}
```

## Data Relationships

### Entity Relationship Diagram

```
System Configuration
├── Security Settings
│   ├── Authentication Configuration
│   └── Persona Definitions
├── Pipeline Configuration
│   └── Task Definitions
├── Context Settings
└── Workspace Settings

Persona
├── Security Context
│   ├── Permissions
│   └── Constraints
├── Capabilities
├── Tools Allowed
└── AI Context

Pipeline
├── Metadata
├── Tasks (Array)
│   ├── Inputs/Outputs
│   ├── Dependencies
│   └── Execution Settings
└── Execution Configuration

Context
├── Metadata
├── Data
└── Configuration
│   ├── Compression
│   └── Persistence

Workspace
├── Resource Limits
├── Mount Points
└── Process Information

Message
├── Headers
├── Payload
└── Routing Information

Validation
├── Rules
├── Results
└── Reports
```

### Data Flow Patterns

1. **Pipeline Execution Flow**:
   Pipeline → Task(s) → Context → Workspace → Claude Adapter → Response → Context

2. **Security Enforcement Flow**:
   Request → Persona Validation → Permission Check → Constraint Application → Execution

3. **Context Management Flow**:
   Context Creation → Data Population → Compaction → Persistence → Retrieval → Cleanup

4. **Workspace Lifecycle Flow**:
   Workspace Creation → Resource Allocation → Process Execution → Cleanup → Deletion

## Performance Considerations

### Data Size Estimates

| Entity          | Average Size | Maximum Size | Storage Location |
|-----------------|-------------|-------------|------------------|
| Configuration   | 10KB        | 100KB       | Memory/Cache     |
| Persona         | 5KB         | 50KB        | Memory           |
| Pipeline        | 50KB        | 1MB         | Memory + Disk    |
| Context         | 1MB         | 100MB       | Memory + Disk    |
| Workspace       | Variable    | 10GB        | Disk             |
| Message         | 1KB         | 10MB        | Memory/Queue     |

### Access Patterns

| Entity          | Read Frequency | Write Frequency | Access Type |
|-----------------|---------------|----------------|------------|
| Configuration   | Low           | Very Low       | Sequential |
| Persona         | Medium        | Low            | Random     |
| Pipeline        | Medium        | Medium         | Sequential |
| Context         | High          | High           | Random     |
| Workspace       | Medium        | High           | Sequential |
| Message         | Very High     | Very High      | Queue      |

### Optimization Strategies

1. **Context Optimization**:
   - LRU cache for frequently accessed contexts
   - Background compaction for large contexts
   - Tiered storage (memory → SSD → disk)

2. **Workspace Optimization**:
   - Lazy allocation of workspace resources
   - Copy-on-write for shared data
   - Background cleanup processes

3. **Message Optimization**:
   - Binary serialization for high-throughput paths
   - Message batching for bulk operations
   - Zero-copy message passing where possible

This data model provides the foundation for implementing the Wave system with clear interfaces, type safety, and optimal performance characteristics.
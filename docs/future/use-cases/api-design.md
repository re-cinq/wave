---
title: API Design
description: Design and validate APIs with schema generation, documentation, and contract testing
---

# API Design

<div class="use-case-meta">
  <span class="complexity-badge intermediate">Intermediate</span>
  <span class="category-badge">Documentation</span>
</div>

Design and validate APIs with schema generation, documentation, and contract testing. This pipeline helps you create well-documented, consistent APIs with OpenAPI specifications.

## Prerequisites

- Wave installed and initialized (`wave init`)
- Understanding of REST/HTTP API design principles
- Experience with [documentation-generation](./documentation-generation) pipeline (recommended)
- Familiarity with OpenAPI/Swagger specifications

## Quick Start

```bash
wave run api-design "design REST API for user management (CRUD operations)"
```

Expected output:

```
[10:00:01] started   requirements      (navigator)              Starting step
[10:00:28] completed requirements      (navigator)   27s   2.4k Requirements complete
[10:00:29] started   design            (philosopher)            Starting step
[10:01:15] completed design            (philosopher)  46s   5.8k Design complete
[10:01:16] started   validate          (auditor)                Starting step
[10:01:42] completed validate          (auditor)     26s   2.1k Validation complete

Pipeline api-design completed in 101s
Artifacts: output/api-spec.yaml
```

## Complete Pipeline

Save the following YAML to `.wave/pipelines/api-design.yaml`:

<div v-pre>

```yaml
kind: WavePipeline
metadata:
  name: api-design
  description: "Design and document APIs with OpenAPI specification"

input:
  source: cli

steps:
  - id: requirements
    persona: navigator
    memory:
      strategy: fresh
    workspace:
      mount:
        - source: ./
          target: /src
          mode: readonly
    exec:
      type: prompt
      source: |
        Gather API requirements for: {{ input }}

        Analyze:
        1. What resources need to be exposed?
        2. What operations are needed (CRUD, custom actions)?
        3. What data models are involved?
        4. What are the authentication/authorization requirements?
        5. What are the performance requirements?
        6. Are there existing APIs to maintain consistency with?

        Output as JSON:
        {
          "resources": [{"name": "", "description": "", "operations": []}],
          "data_models": [{"name": "", "fields": []}],
          "authentication": "",
          "rate_limits": {},
          "existing_patterns": []
        }
    output_artifacts:
      - name: requirements
        path: output/api-requirements.json
        type: json

  - id: design
    persona: philosopher
    dependencies: [requirements]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: requirements
          artifact: requirements
          as: requirements
    exec:
      type: prompt
      source: |
        Design the API based on requirements: {{ input }}

        Create an OpenAPI 3.0 specification including:
        1. Info and server configuration
        2. Paths for all endpoints
        3. Request/response schemas
        4. Authentication schemes
        5. Error responses
        6. Examples for each endpoint

        Follow best practices:
        - RESTful naming conventions
        - Consistent response structure
        - Meaningful HTTP status codes
        - Pagination for list endpoints
        - Versioning strategy
        - HATEOAS links where appropriate
    output_artifacts:
      - name: spec
        path: output/api-spec.yaml
        type: yaml

  - id: validate
    persona: auditor
    dependencies: [design]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: requirements
          artifact: requirements
          as: requirements
        - step: design
          artifact: spec
          as: api_spec
    exec:
      type: prompt
      source: |
        Validate the API design:

        Check:
        1. All requirements addressed?
        2. OpenAPI spec is valid?
        3. Naming conventions consistent?
        4. Error handling comprehensive?
        5. Security considerations addressed?
        6. Pagination implemented for collections?
        7. Versioning strategy clear?
        8. Documentation complete?

        Output: list of issues or "APPROVED"
    output_artifacts:
      - name: validation
        path: output/api-validation.md
        type: markdown
```

</div>

## Expected Outputs

The pipeline produces three artifacts:

| Artifact | Path | Description |
|----------|------|-------------|
| `requirements` | `output/api-requirements.json` | Structured API requirements |
| `spec` | `output/api-spec.yaml` | OpenAPI 3.0 specification |
| `validation` | `output/api-validation.md` | Design review and validation |

### Example Output

The pipeline produces `output/api-spec.yaml`:

<div v-pre>

```yaml
openapi: 3.0.3
info:
  title: User Management API
  description: REST API for user management operations
  version: 1.0.0
  contact:
    name: API Support
    email: api@example.com

servers:
  - url: https://api.example.com/v1
    description: Production server
  - url: https://staging-api.example.com/v1
    description: Staging server

paths:
  /users:
    get:
      summary: List all users
      description: Retrieve a paginated list of users
      operationId: listUsers
      tags:
        - Users
      parameters:
        - name: page
          in: query
          schema:
            type: integer
            default: 1
        - name: limit
          in: query
          schema:
            type: integer
            default: 20
            maximum: 100
        - name: sort
          in: query
          schema:
            type: string
            enum: [created_at, name, email]
      responses:
        '200':
          description: Successful response
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/UserList'
        '401':
          $ref: '#/components/responses/Unauthorized'

    post:
      summary: Create a new user
      description: Create a new user account
      operationId: createUser
      tags:
        - Users
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateUserRequest'
      responses:
        '201':
          description: User created
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/User'
        '400':
          $ref: '#/components/responses/BadRequest'
        '409':
          $ref: '#/components/responses/Conflict'

  /users/{userId}:
    get:
      summary: Get a user by ID
      operationId: getUser
      tags:
        - Users
      parameters:
        - $ref: '#/components/parameters/UserId'
      responses:
        '200':
          description: Successful response
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/User'
        '404':
          $ref: '#/components/responses/NotFound'

    put:
      summary: Update a user
      operationId: updateUser
      tags:
        - Users
      parameters:
        - $ref: '#/components/parameters/UserId'
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/UpdateUserRequest'
      responses:
        '200':
          description: User updated
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/User'
        '404':
          $ref: '#/components/responses/NotFound'

    delete:
      summary: Delete a user
      operationId: deleteUser
      tags:
        - Users
      parameters:
        - $ref: '#/components/parameters/UserId'
      responses:
        '204':
          description: User deleted
        '404':
          $ref: '#/components/responses/NotFound'

components:
  schemas:
    User:
      type: object
      required:
        - id
        - email
        - name
        - created_at
      properties:
        id:
          type: string
          format: uuid
          example: "123e4567-e89b-12d3-a456-426614174000"
        email:
          type: string
          format: email
          example: "user@example.com"
        name:
          type: string
          example: "John Doe"
        role:
          type: string
          enum: [admin, user, guest]
          default: user
        created_at:
          type: string
          format: date-time
        updated_at:
          type: string
          format: date-time

    UserList:
      type: object
      properties:
        data:
          type: array
          items:
            $ref: '#/components/schemas/User'
        pagination:
          $ref: '#/components/schemas/Pagination'

    CreateUserRequest:
      type: object
      required:
        - email
        - name
        - password
      properties:
        email:
          type: string
          format: email
        name:
          type: string
          minLength: 1
          maxLength: 100
        password:
          type: string
          minLength: 8
        role:
          type: string
          enum: [admin, user, guest]

    UpdateUserRequest:
      type: object
      properties:
        name:
          type: string
        role:
          type: string
          enum: [admin, user, guest]

    Pagination:
      type: object
      properties:
        page:
          type: integer
        limit:
          type: integer
        total:
          type: integer
        total_pages:
          type: integer

    Error:
      type: object
      required:
        - code
        - message
      properties:
        code:
          type: string
        message:
          type: string
        details:
          type: object

  parameters:
    UserId:
      name: userId
      in: path
      required: true
      schema:
        type: string
        format: uuid

  responses:
    BadRequest:
      description: Bad request
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/Error'
    Unauthorized:
      description: Unauthorized
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/Error'
    NotFound:
      description: Resource not found
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/Error'
    Conflict:
      description: Resource conflict
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/Error'

  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT

security:
  - bearerAuth: []
```

</div>

## Customization

### Design specific API type

```bash
wave run api-design "design GraphQL API for product catalog"
```

```bash
wave run api-design "design WebSocket API for real-time notifications"
```

### Include implementation

Add a code generation step:

<div v-pre>

```yaml
- id: implement
  persona: craftsman
  dependencies: [design]
  memory:
    inject_artifacts:
      - step: design
        artifact: spec
        as: api_spec
  exec:
    source: |
      Generate Go handler stubs from the OpenAPI spec.
      Include:
      - Handler interfaces
      - Request/response types
      - Validation middleware
      - Router configuration
  output_artifacts:
    - name: handlers
      path: output/handlers.go
      type: code
```

</div>

### Add contract tests

<div v-pre>

```yaml
- id: contract-tests
  persona: craftsman
  dependencies: [design]
  memory:
    inject_artifacts:
      - step: design
        artifact: spec
        as: api_spec
  exec:
    source: |
      Generate contract tests from the OpenAPI spec.
      Test each endpoint for:
      - Valid requests
      - Invalid requests (validation errors)
      - Error responses
      - Edge cases
  handover:
    contract:
      type: test_suite
      command: "go test ./api/... -v"
      must_pass: true
  output_artifacts:
    - name: tests
      path: output/api_test.go
      type: code
```

</div>

## API Design Best Practices

### Naming Conventions

- Use plural nouns for collections: `/users`, `/orders`
- Use kebab-case for multi-word paths: `/user-preferences`
- Use query params for filtering: `/users?role=admin`

### HTTP Methods

| Method | Purpose | Example |
|--------|---------|---------|
| GET | Retrieve resource(s) | `GET /users/123` |
| POST | Create resource | `POST /users` |
| PUT | Replace resource | `PUT /users/123` |
| PATCH | Partial update | `PATCH /users/123` |
| DELETE | Remove resource | `DELETE /users/123` |

### Response Codes

| Code | Meaning | Use Case |
|------|---------|----------|
| 200 | OK | Successful GET, PUT, PATCH |
| 201 | Created | Successful POST |
| 204 | No Content | Successful DELETE |
| 400 | Bad Request | Validation error |
| 401 | Unauthorized | Missing/invalid auth |
| 403 | Forbidden | Insufficient permissions |
| 404 | Not Found | Resource doesn't exist |
| 409 | Conflict | Duplicate resource |
| 429 | Too Many Requests | Rate limited |
| 500 | Server Error | Unexpected error |

## Related Use Cases

- [Documentation Generation](./documentation-generation) - Generate API docs
- [Test Generation](/use-cases/test-generation) - Generate API tests
- [Code Review](/use-cases/code-review) - Review API implementations

## Next Steps

- [Concepts: Contracts](/concepts/contracts) - Validate API outputs
- [Concepts: Artifacts](/concepts/artifacts) - Pass specs between steps

<style>
.use-case-meta {
  display: flex;
  gap: 8px;
  margin-bottom: 24px;
}
.complexity-badge {
  padding: 4px 12px;
  font-size: 12px;
  font-weight: 600;
  border-radius: 12px;
  text-transform: uppercase;
}
.complexity-badge.beginner {
  background: #dcfce7;
  color: #166534;
}
.complexity-badge.intermediate {
  background: #fef3c7;
  color: #92400e;
}
.complexity-badge.advanced {
  background: #fee2e2;
  color: #991b1b;
}
.category-badge {
  padding: 4px 12px;
  font-size: 12px;
  font-weight: 500;
  border-radius: 12px;
  background: var(--vp-c-brand-soft);
  color: var(--vp-c-brand-1);
}
</style>

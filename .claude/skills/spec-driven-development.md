---
description: Expert specification-driven development including TDD/BDD integration, living documentation, specification-to-code workflows, and validation strategies
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Outline

You are a specification-driven development expert specializing in behavior-driven development, test-driven development, living documentation, and specification-to-code workflows. Use this skill when the user needs help with:

- Writing specifications and executable tests
- Implementing TDD/BDD methodologies
- Creating living documentation systems
- Specification-to-code automation
- Validation and testing strategies
- Requirements traceability
- Acceptance test-driven development

## Core Specification-Driven Development Expertise

### 1. Behavior-Driven Development (BDD)

#### Gherkin Syntax and Feature Files
```gherkin
# user-registration.feature
Feature: User Registration
  As a new user
  I want to create an account
  So that I can access the application

  Background:
    Given the user registration system is available
    And the email service is operational

  Scenario: Successful user registration
    Given I am on the registration page
    When I enter valid user details:
      | field    | value               |
      | email    | test@example.com    |
      | password | SecurePass123!      |
      | name     | John Doe           |
    And I click the "Register" button
    Then I should see a success message
    And I should receive a confirmation email
    And my account should be created in the system

  Scenario: Registration with invalid email
    Given I am on the registration page
    When I enter user details with an invalid email:
      | field    | value          |
      | email    | invalid-email  |
      | password | SecurePass123! |
      | name     | John Doe       |
    And I click the "Register" button
    Then I should see an email validation error
    And my account should not be created

  Scenario Outline: Password validation
    Given I am on the registration page
    When I enter user details with password "<password>"
    And I click the "Register" button
    Then I should see the message "<error_message>"

    Examples:
      | password     | error_message                   |
      | short        | Password must be at least 8 characters |
      | nopunctuation | Password must contain special character |
      | nonumber     | Password must contain at least one number |
```

#### BDD Test Implementation
```python
# Python Behave step definitions
from behave import given, when, then
from pages.registration_page import RegistrationPage
from services.user_service import UserService
from utils.email_validator import EmailValidator

@given("I am on the registration page")
def step_impl(context):
    context.registration_page = RegistrationPage()
    context.registration_page.navigate()

@when('I enter valid user details:')
def step_impl(context):
    user_data = {row['field']: row['value'] for row in context.table}
    for field, value in user_data.items():
        context.registration_page.enter_field(field, value)

@when('I click the "{button}" button')
def step_impl(context, button):
    context.registration_page.click_button(button)

@then("I should see a success message")
def step_impl(context):
    assert context.registration_page.get_success_message() is not None

@then("I should receive a confirmation email")
def step_impl(context):
    # In a real implementation, this would check email service
    context.email_sent = True
    assert context.email_sent

@then("my account should be created in the system")
def step_impl(context):
    user_service = UserService()
    created_user = user_service.get_user_by_email("test@example.com")
    assert created_user is not None
```

### 2. Test-Driven Development (TDD)

#### Red-Green-Refactor Cycle
```go
// Go TDD example for string calculator

// 1. RED - Write failing test
func TestStringCalculator_Add_EmptyString_ReturnsZero(t *testing.T) {
    calculator := StringCalculator{}
    result, err := calculator.Add("")
    assert.NoError(t, err)
    assert.Equal(t, 0, result)
}

// 2. GREEN - Write minimum code to pass
type StringCalculator struct{}

func (sc StringCalculator) Add(numbers string) (int, error) {
    if numbers == "" {
        return 0, nil
    }
    return 0, errors.New("not implemented")
}

// 3. Add more tests
func TestStringCalculator_Add_SingleNumber_ReturnsNumber(t *testing.T) {
    calculator := StringCalculator{}
    result, err := calculator.Add("1")
    assert.NoError(t, err)
    assert.Equal(t, 1, result)
}

func TestStringCalculator_Add_TwoNumbers_ReturnsSum(t *testing.T) {
    calculator := StringCalculator{}
    result, err := calculator.Add("1,2")
    assert.NoError(t, err)
    assert.Equal(t, 3, result)
}

// 4. Refactor to improve implementation
func (sc StringCalculator) Add(numbers string) (int, error) {
    if numbers == "" {
        return 0, nil
    }
    
    parts := strings.Split(numbers, ",")
    sum := 0
    for _, part := range parts {
        num, err := strconv.Atoi(strings.TrimSpace(part))
        if err != nil {
            return 0, fmt.Errorf("invalid number: %s", part)
        }
        sum += num
    }
    return sum, nil
}
```

#### Test-Driven API Development
```java
// Java TDD example for REST API
@WebMvcTest(UserController.class)
public class UserControllerTest {
    
    @Autowired
    private MockMvc mockMvc;
    
    @MockBean
    private UserService userService;
    
    @Test
    public void getUser_WhenUserExists_ReturnsUser() throws Exception {
        // Arrange
        User user = new User("1", "test@example.com", "Test User");
        when(userService.getUser("1")).thenReturn(Optional.of(user));
        
        // Act & Assert
        mockMvc.perform(get("/api/users/1"))
            .andExpect(status().isOk())
            .andExpect(jsonPath("$.id").value("1"))
            .andExpect(jsonPath("$.email").value("test@example.com"))
            .andExpect(jsonPath("$.name").value("Test User"));
    }
    
    @Test
    public void getUser_WhenUserNotFound_Returns404() throws Exception {
        // Arrange
        when(userService.getUser("1")).thenReturn(Optional.empty());
        
        // Act & Assert
        mockMvc.perform(get("/api/users/1"))
            .andExpect(status().isNotFound());
    }
    
    @Test
    public void createUser_WithValidData_ReturnsCreatedUser() throws Exception {
        // Arrange
        UserCreateRequest request = new UserCreateRequest("test@example.com", "Test User");
        User createdUser = new User("1", "test@example.com", "Test User");
        when(userService.createUser(any())).thenReturn(createdUser);
        
        // Act & Assert
        mockMvc.perform(post("/api/users")
                .contentType(MediaType.APPLICATION_JSON)
                .content(objectMapper.writeValueAsString(request)))
            .andExpect(status().isCreated())
            .andExpect(jsonPath("$.id").value("1"))
            .andExpect(jsonPath("$.email").value("test@example.com"));
    }
}
```

### 3. Living Documentation

#### Documentation as Code
```python
# Python living documentation generator
class DocumentationGenerator:
    def __init__(self, feature_files_path: str, output_path: str):
        self.feature_files_path = feature_files_path
        self.output_path = output_path
    
    def generate_living_docs(self):
        """Generate HTML documentation from Gherkin feature files"""
        features = self._parse_features()
        html_content = self._render_html(features)
        self._save_documentation(html_content)
    
    def _parse_features(self) -> List[Dict]:
        """Parse Gherkin features and extract test data"""
        features = []
        for feature_file in glob.glob(f"{self.feature_files_path}/*.feature"):
            with open(feature_file, 'r') as f:
                feature = self._parse_feature_content(f.read())
                features.append(feature)
        return features
    
    def _parse_feature_content(self, content: str) -> Dict:
        """Parse individual feature file"""
        lines = content.split('\n')
        feature = {
            'title': '',
            'description': '',
            'scenarios': [],
            'background': None
        }
        
        current_scenario = None
        
        for line in lines:
            line = line.strip()
            if line.startswith('Feature:'):
                feature['title'] = line.replace('Feature:', '').strip()
            elif line.startswith('  Description:'):
                feature['description'] = line.replace('  Description:', '').strip()
            elif line.startswith('  Scenario:'):
                if current_scenario:
                    feature['scenarios'].append(current_scenario)
                current_scenario = {
                    'title': line.replace('  Scenario:', '').strip(),
                    'steps': []
                }
            elif line.startswith('  Background:'):
                feature['background'] = {'steps': []}
                current_scenario = feature['background']
            elif line.startswith('    Given') or line.startswith('    When') or line.startswith('    Then'):
                if current_scenario:
                    current_scenario['steps'].append(line.strip())
        
        if current_scenario and current_scenario not in feature['scenarios']:
            feature['scenarios'].append(current_scenario)
        
        return feature
    
    def _render_html(self, features: List[Dict]) -> str:
        """Render HTML documentation"""
        html = """
<!DOCTYPE html>
<html>
<head>
    <title>Living Documentation</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .feature { margin-bottom: 30px; border: 1px solid #ddd; padding: 20px; }
        .scenario { margin: 15px 0; padding: 10px; background-color: #f9f9f9; }
        .step { margin: 5px 0; font-family: monospace; }
        .given { color: #0066cc; }
        .when { color: #ff6600; }
        .then { color: #009900; }
    </style>
</head>
<body>
    <h1>Living Documentation</h1>
"""
        
        for feature in features:
            html += f"""
    <div class="feature">
        <h2>{feature['title']}</h2>
        <p>{feature['description']}</p>
"""
            
            for scenario in feature['scenarios']:
                html += f"""
        <div class="scenario">
            <h3>{scenario['title']}</h3>
"""
                for step in scenario['steps']:
                    css_class = step.lower().split()[0] if step.split() else ''
                    html += f'            <div class="step {css_class}">{step}</div>\n'
                html += "        </div>\n"
            
            html += "    </div>\n"
        
        html += "</body>\n</html>"
        return html
```

### 4. Specification-to-Code Workflows

#### Code Generation from Specifications
```typescript
// TypeScript code generator from OpenAPI specs
interface ApiEndpoint {
  path: string;
  method: string;
  parameters: Parameter[];
  responses: Response[];
  requestBody?: RequestBody;
}

interface Parameter {
  name: string;
  in: 'path' | 'query' | 'header';
  required: boolean;
  type: string;
}

class ApiClientGenerator {
  generateClient(spec: OpenApiSpec): string {
    const endpoints = this.extractEndpoints(spec);
    return `
// Generated API Client from OpenAPI Specification
export class ApiClient {
  private baseUrl: string;
  
  constructor(baseUrl: string = '${spec.servers[0].url}') {
    this.baseUrl = baseUrl;
  }
  
  ${endpoints.map(endpoint => this.generateEndpointMethod(endpoint)).join('\n\n')}
}`;
  }
  
  private generateEndpointMethod(endpoint: ApiEndpoint): string {
    const methodName = this.getMethodName(endpoint);
    const parameters = this.generateParameters(endpoint);
    const urlPath = this.buildUrlPath(endpoint);
    
    return `  async ${methodName}(${parameters}): Promise<${this.getResponseType(endpoint)}> {
    const url = \`\${this.baseUrl}${urlPath}\`;
    const response = await fetch(url, {
      method: '${endpoint.method.toUpperCase()}',
      headers: this.getHeaders(),
      ${endpoint.requestBody ? `body: JSON.stringify(body),` : ''}
    });
    
    if (!response.ok) {
      throw new Error(\`API request failed: \${response.status}\`);
    }
    
    return response.json();
  }`;
  }
  
  private getMethodName(endpoint: ApiEndpoint): string {
    const path = endpoint.path.replace(/[{}]/g, '').replace(/\//g, '');
    return endpoint.method.toLowerCase() + path.charAt(0).toUpperCase() + path.slice(1);
  }
}
```

#### Contract Testing with Pact
```javascript
// JavaScript Pact consumer test
const { Pact } = require('@pact-foundation/pact');
const { expect } = require('chai');

describe('User API', () => {
  const provider = new Pact({
    consumer: 'web-app',
    provider: 'user-service',
    port: 1234,
    log: './pact/logs/pact.log',
    dir: './pact/pacts',
    logLevel: 'INFO',
  });

  before(() => provider.setup());
  after(() => provider.finalize());

  describe('Get User', () => {
    beforeEach(() =>
      provider.addInteraction({
        state: 'user with ID 1 exists',
        uponReceiving: 'a request for user 1',
        withRequest: {
          method: 'GET',
          path: '/api/users/1',
          headers: {
            Accept: 'application/json',
          },
        },
        willRespondWith: {
          status: 200,
          headers: {
            'Content-Type': 'application/json; charset=utf-8',
          },
          body: {
            id: 1,
            name: 'John Doe',
            email: 'john@example.com',
          },
        },
      })
    );

    it('returns the user', async () => {
      const response = await fetch(`${provider.url}/api/users/1`, {
        headers: { Accept: 'application/json' },
      });
      
      const user = await response.json();
      expect(user.id).to.equal(1);
      expect(user.name).to.equal('John Doe');
      expect(user.email).to.equal('john@example.com');
    });
  });
});
```

### 5. Validation and Testing Strategies

#### Property-Based Testing
```rust
// Rust property-based testing with proptest
use proptest::prelude::*;

proptest! {
    #[test]
    fn test_string_calculator_properties(s in "\\PC*") {
        let calculator = StringCalculator::new();
        
        // Property: Empty string returns 0
        if s.is_empty() {
            prop_assert_eq!(calculator.add(&s).unwrap(), 0);
        }
        
        // Property: Single number returns that number
        if let Ok(num) = s.parse::<i32>() {
            prop_assert_eq!(calculator.add(&s).unwrap(), num);
        }
    }
    
    #[test]
    fn test_commutative_property(nums in prop::collection::vec(0..100i32, 2..=5)) {
        let calculator = StringCalculator::new();
        let numbers_str: Vec<String> = nums.iter().map(|n| n.to_string()).collect();
        let joined = numbers_str.join(",");
        
        let result1 = calculator.add(&joined).unwrap();
        let result2 = calculator.add(&numbers_str.iter().rev().cloned().collect::<Vec<_>>().join(",")).unwrap();
        
        prop_assert_eq!(result1, result2);
    }
}
```

#### Mutation Testing
```python
# Python mutation testing configuration
# mutmut_config.py
def setup_mutation_tests():
    """Configure mutation testing for specification-driven development"""
    
    # Test patterns to exclude from mutation
    exclude_patterns = [
        "test_*_spec*",  # BDD specification tests
        "*_bdd_test*",   # Behavior-driven tests
        "feature_*",      # Feature tests
    ]
    
    # Mutation operators
    mutation_operators = [
        "arithmetic_operator_deletion",
        "boolean_replacement",
        "conditional_replacement",
        "constant_replacement",
        "return_value_replacement",
    ]
    
    return {
        "exclude_patterns": exclude_patterns,
        "mutation_operators": mutation_operators,
        "test_command": "pytest tests/ --tb=short",
        "coverage_threshold": 80.0,
    }

# Run mutation tests
def run_mutation_tests():
    """Execute mutation testing to ensure test quality"""
    import subprocess
    import json
    
    config = setup_mutation_tests()
    
    # Run mutmut
    result = subprocess.run([
        "mutmut", "run",
        "--paths-to-mutate=src/",
        "--tests-at-depths",
        "--runner=python -m pytest -x",
        f"--coverage-threshold={config['coverage_threshold']}"
    ], capture_output=True, text=True)
    
    if result.returncode != 0:
        print("Mutation tests failed:")
        print(result.stdout)
        print(result.stderr)
        return False
    
    return True
```

### 6. Requirements Traceability

#### Traceability Matrix
```go
// Go requirements traceability implementation
type Requirement struct {
    ID          string
    Title       string
    Description string
    Priority    string
    Tests       []TestReference
    Code        []CodeReference
}

type TestReference struct {
    Type        string  // "unit", "integration", "e2e"
    Path        string
    Name        string
    LastRun     time.Time
    Status      string
}

type CodeReference struct {
    Type string  // "function", "class", "module"
    Path string
    Name string
}

type TraceabilityMatrix struct {
    requirements []Requirement
}

func (tm *TraceabilityMatrix) AddRequirement(req Requirement) {
    tm.requirements = append(tm.requirements, req)
}

func (tm *TraceabilityMatrix) GetRequirementByID(id string) *Requirement {
    for i := range tm.requirements {
        if tm.requirements[i].ID == id {
            return &tm.requirements[i]
        }
    }
    return nil
}

func (tm *TraceabilityMatrix) GenerateReport() string {
    var report strings.Builder
    
    report.WriteString("# Requirements Traceability Matrix\n\n")
    report.WriteString("| Requirement ID | Title | Test Coverage | Code Coverage | Status |\n")
    report.WriteString("|---------------|-------|---------------|---------------|--------|\n")
    
    for _, req := range tm.requirements {
        testCoverage := len(req.Tests)
        codeCoverage := len(req.Code)
        status := tm.calculateStatus(req)
        
        report.WriteString(fmt.Sprintf("| %s | %s | %d | %d | %s |\n",
            req.ID, req.Title, testCoverage, codeCoverage, status))
    }
    
    return report.String()
}

func (tm *TraceabilityMatrix) calculateStatus(req Requirement) string {
    if len(req.Tests) == 0 {
        return "No Tests"
    }
    
    allPassed := true
    for _, test := range req.Tests {
        if test.Status != "passed" {
            allPassed = false
            break
        }
    }
    
    if allPassed {
        return "Covered"
    }
    return "Failing"
}
```

## Best Practices and Patterns

### 1. Specification Quality
- Write specifications in business language
- Keep scenarios independent and atomic
- Use examples to clarify complex rules
- Review specifications with domain experts

### 2. Test Organization
- Separate unit, integration, and end-to-end tests
- Use descriptive test names that tell a story
- Follow AAA pattern (Arrange, Act, Assert)
- Keep tests focused on single behavior

### 3. Documentation Maintenance
- Auto-generate documentation from tests
- Keep documentation synchronized with code
- Include examples and usage patterns
- Update documentation with each feature change

### 4. Continuous Integration
- Run all specification tests on each commit
- Generate and verify documentation builds
- Run mutation tests regularly
- Monitor test coverage and quality metrics

### 5. Common Pitfalls
- ** brittle Tests**: Avoid over-specifying implementation details
- **Lost Traceability**: Maintain links between requirements and code
- **Stale Documentation**: Automate documentation updates
- **Over-testing**: Focus on business-critical scenarios

## When to Use Specification-Driven Development

### Ideal Scenarios
- Complex business domains with many rules
- Requirements that evolve frequently
- Teams with domain expert collaboration
- Regulatory compliance requirements
- Long-lived applications requiring maintenance

### Less Suitable Scenarios
- Simple CRUD applications
- Proof-of-concept prototypes
- Performance-critical low-level systems
- Solo projects with stable requirements

This specification-driven development skill provides comprehensive expertise in creating robust, maintainable systems where specifications drive development, tests provide living documentation, and quality is built into the process from the beginning.
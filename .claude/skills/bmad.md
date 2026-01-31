---
description: Expert BMAD (Breakthrough Method for Agile AI-Driven Development) implementation including role-based agent specialization, structured workflows, living artifacts, and scale-adaptive processes
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Outline

You are a BMAD (Breakthrough Method for Agile AI-Driven Development) expert specializing in AI-enhanced development workflows, role-based agent specialization, and structured collaborative development processes. Use this skill when the user needs help with:

- Implementing BMAD methodology and workflows
- Setting up role-based agent teams (PM, Architect, Dev, QA)
- Creating structured development processes (discovery → planning → execution → verification)
- Managing living artifacts and governance
- Scaling processes for different team sizes
- AI-human collaboration optimization
- Continuous improvement and feedback loops

## Core BMAD Expertise

### 1. BMAD Fundamentals

#### Breakthrough Method Principles
- **AI-Enhanced Agility**: Leverage AI for rapid iteration and adaptation
- **Role Specialization**: Clear agent roles with defined responsibilities
- **Structured Flexibility**: Balanced process with room for creativity
- **Living Documentation**: Artifacts that evolve with the project
- **Continuous Verification**: Ongoing validation of decisions and outcomes
- **Scale-Adaptive Processes**: Methods that scale with team and project size

#### Core BMAD Workflow
```python
# BMAD workflow orchestration
class BMADWorkflow:
    def __init__(self, team_size: int, project_complexity: str):
        self.team_size = team_size
        self.project_complexity = project_complexity
        self.current_phase = "discovery"
        self.agents = self._initialize_agents()
        self.artifacts = ArtifactManager()
        
    def _initialize_agents(self) -> Dict[str, Agent]:
        """Initialize role-based AI agents based on team size"""
        agents = {
            "project_manager": ProjectManagerAgent(),
            "architect": ArchitectAgent(),
            "developer": DeveloperAgent(),
            "qa_agent": QAAgent()
        }
        
        # Scale agents based on team size
        if self.team_size > 5:
            agents["senior_developer"] = SeniorDeveloperAgent()
            agents["devops_agent"] = DevOpsAgent()
        
        if self.project_complexity in ["high", "critical"]:
            agents["security_agent"] = SecurityAgent()
            agents["performance_agent"] = PerformanceAgent()
        
        return agents
    
    async def execute_phase(self, phase: str, inputs: Dict) -> Dict:
        """Execute specific BMAD phase with appropriate agents"""
        self.current_phase = phase
        
        if phase == "discovery":
            return await self._discovery_phase(inputs)
        elif phase == "planning":
            return await self._planning_phase(inputs)
        elif phase == "execution":
            return await self._execution_phase(inputs)
        elif phase == "verification":
            return await self._verification_phase(inputs)
        else:
            raise ValueError(f"Unknown phase: {phase}")
    
    async def _discovery_phase(self, inputs: Dict) -> Dict:
        """Discovery phase: requirements gathering and analysis"""
        # Project Manager leads discovery
        pm_results = await self.agents["project_manager"].facilitate_discovery(inputs)
        
        # Architect analyzes technical implications
        tech_analysis = await self.agents["architect"].analyze_requirements(pm_results)
        
        # QA identifies testing requirements
        test_requirements = await self.agents["qa_agent"].identify_testing_needs(pm_results)
        
        # Create living specification
        specification = self.artifacts.create_specification({
            "requirements": pm_results,
            "technical_analysis": tech_analysis,
            "testing_requirements": test_requirements,
            "timestamp": datetime.now().isoformat()
        })
        
        return {
            "specification": specification,
            "next_phase": "planning",
            "agents_involved": ["project_manager", "architect", "qa_agent"]
        }
```

### 2. Role-Based Agent Specialization

#### Project Manager Agent
```typescript
interface ProjectManagerCapabilities {
  discoveryFacilitation: boolean;
  stakeholderManagement: boolean;
  riskAssessment: boolean;
  progressTracking: boolean;
  resourcePlanning: boolean;
}

class ProjectManagerAgent extends BaseAgent {
  private capabilities: ProjectManagerCapabilities;
  
  constructor() {
    super();
    this.capabilities = {
      discoveryFacilitation: true,
      stakeholderManagement: true,
      riskAssessment: true,
      progressTracking: true,
      resourcePlanning: true
    };
  }
  
  async facilitateDiscovery(inputs: DiscoveryInputs): Promise<DiscoveryResults> {
    const session = await this.createDiscoverySession(inputs);
    
    // AI-powered stakeholder analysis
    const stakeholderAnalysis = await this.analyzeStakeholders(inputs.stakeholders);
    
    // Automated requirement extraction
    const requirements = await this.extractRequirements(inputs.sources);
    
    // Risk identification and assessment
    const risks = await this.assessRisks(requirements, stakeholderAnalysis);
    
    return {
      requirements: this.prioritizeRequirements(requirements),
      stakeholders: stakeholderAnalysis,
      risks: this.mitigateRisks(risks),
      assumptions: this.identifyAssumptions(inputs),
      dependencies: this.identifyDependencies(requirements)
    };
  }
  
  private async extractRequirements(sources: DataSource[]): Promise<Requirement[]> {
    const requirements: Requirement[] = [];
    
    for (const source of sources) {
      // Use NLP to extract requirements from various sources
      const extracted = await this.nlpProcessor.extractRequirements(source);
      
      // Categorize and validate requirements
      const validated = await this.validateRequirements(extracted);
      requirements.push(...validated);
    }
    
    return this.deduplicateRequirements(requirements);
  }
  
  private async assessRisks(requirements: Requirement[], stakeholders: Stakeholder[]): Promise<Risk[]> {
    const risks: Risk[] = [];
    
    // Technical risk assessment
    const techRisks = await this.assessTechnicalRisks(requirements);
    risks.push(...techRisks);
    
    // Business risk assessment
    const businessRisks = await this.assessBusinessRisks(requirements, stakeholders);
    risks.push(...businessRisks);
    
    // Resource risk assessment
    const resourceRisks = await this.assessResourceRisks(requirements);
    risks.push(...resourceRisks);
    
    return this.prioritizeRisks(risks);
  }
}
```

#### Architect Agent
```go
// Go architect agent implementation
type ArchitectAgent struct {
    knowledgeBase    *KnowledgeBase
    patternLibrary   *PatternLibrary
    decisionLogger   *DecisionLogger
}

type ArchitectCapabilities struct {
    SystemDesign     bool
    TechnologySelection bool
    PatternMatching  bool
    IntegrationDesign bool
    ScalabilityPlanning bool
}

func (aa *ArchitectAgent) AnalyzeRequirements(discoveryResults DiscoveryResults) (*TechnicalAnalysis, error) {
    analysis := &TechnicalAnalysis{
        Requirements: discoveryResults.Requirements,
        Constraints:  aa.identifyConstraints(discoveryResults),
        Assumptions:  aa.identifyTechnicalAssumptions(discoveryResults),
    }
    
    // System architecture design
    systemDesign, err := aa.designSystemArchitecture(analysis)
    if err != nil {
        return nil, fmt.Errorf("system architecture design failed: %w", err)
    }
    analysis.SystemDesign = systemDesign
    
    // Technology stack selection
    techStack, err := aa.selectTechnologyStack(systemDesign, analysis.Requirements)
    if err != nil {
        return nil, fmt.Errorf("technology stack selection failed: %w", err)
    }
    analysis.TechnologyStack = techStack
    
    // Integration points identification
    integrations := aa.identifyIntegrationPoints(systemDesign, discoveryResults.Dependencies)
    analysis.Integrations = integrations
    
    // Scalability considerations
    scalability := aa.planScalability(systemDesign, techStack)
    analysis.Scalability = scalability
    
    // Log architectural decisions
    aa.decisionLogger.LogDecision(analysis)
    
    return analysis, nil
}

func (aa *ArchitectAgent) designSystemArchitecture(analysis *TechnicalAnalysis) (*SystemArchitecture, error) {
    // Analyze requirements complexity
    complexity := aa.calculateComplexity(analysis.Requirements)
    
    // Select appropriate architectural pattern
    pattern := aa.patternLibrary.MatchPattern(complexity, analysis.Requirements)
    
    // Design system components
    components := aa.designComponents(pattern, analysis.Requirements)
    
    // Define data flow
    dataFlow := aa.defineDataFlow(components, analysis.Requirements)
    
    // Security architecture
    security := aa.designSecurityArchitecture(components, dataFlow)
    
    return &SystemArchitecture{
        Pattern:      pattern,
        Components:   components,
        DataFlow:    dataFlow,
        Security:    security,
        Complexity:  complexity,
    }, nil
}

func (aa *ArchitectAgent) selectTechnologyStack(arch *SystemArchitecture, requirements []Requirement) (*TechnologyStack, error) {
    stack := &TechnologyStack{}
    
    // Backend technology selection
    backendTech, err := aa.selectBackendTechnology(arch, requirements)
    if err != nil {
        return nil, err
    }
    stack.Backend = backendTech
    
    // Frontend technology selection
    frontendTech, err := aa.selectFrontendTechnology(requirements)
    if err != nil {
        return nil, err
    }
    stack.Frontend = frontendTech
    
    // Database selection
    database, err := aa.selectDatabase(arch, requirements)
    if err != nil {
        return nil, err
    }
    stack.Database = database
    
    // Infrastructure and DevOps tools
    infrastructure := aa.selectInfrastructureTools(arch)
    stack.Infrastructure = infrastructure
    
    return stack, nil
}
```

#### Developer Agent
```python
class DeveloperAgent(BaseAgent):
    def __init__(self):
        super().__init__()
        self.code_analyzer = CodeAnalyzer()
        self.test_generator = TestGenerator()
        self.refactor_engine = RefactorEngine()
        
    async def implement_feature(self, feature_spec: FeatureSpecification) -> ImplementationResult:
        """Implement a feature based on specification"""
        
        # Analyze requirements and existing codebase
        analysis = await self.analyze_implementation_context(feature_spec)
        
        # Generate initial code implementation
        initial_code = await self.generate_code(feature_spec, analysis)
        
        # Create comprehensive tests
        tests = await self.generate_tests(feature_spec, initial_code)
        
        # Refactor and optimize code
        optimized_code = await self.refactor_code(initial_code, tests, analysis)
        
        # Verify implementation quality
        quality_metrics = await self.verify_quality(optimized_code, tests)
        
        # Log implementation decisions
        self.log_implementation_decisions(feature_spec, analysis, optimized_code)
        
        return ImplementationResult(
            code=optimized_code,
            tests=tests,
            metrics=quality_metrics,
            documentation=self.generate_documentation(optimized_code, feature_spec)
        )
    
    async def analyze_implementation_context(self, feature_spec: FeatureSpecification) -> ImplementationContext:
        """Analyze the context for feature implementation"""
        
        # Analyze existing codebase structure
        codebase_structure = await self.code_analyzer.analyze_structure()
        
        # Identify impact areas
        impact_areas = await self.identify_impact_areas(feature_spec, codebase_structure)
        
        # Review existing patterns and conventions
        conventions = await self.analyze_conventions(codebase_structure)
        
        # Identify required dependencies
        dependencies = await self.analyze_dependencies(feature_spec, impact_areas)
        
        return ImplementationContext(
            codebase_structure=codebase_structure,
            impact_areas=impact_areas,
            conventions=conventions,
            dependencies=dependencies
        )
    
    async def generate_code(self, feature_spec: FeatureSpecification, context: ImplementationContext) -> CodeArtifact:
        """Generate initial code implementation"""
        
        # Break down feature into components
        components = self.decompose_feature(feature_spec)
        
        code_artifacts = []
        
        for component in components:
            # Generate code for each component
            component_code = await self.generate_component_code(component, context)
            
            # Apply conventions and patterns
            formatted_code = self.apply_conventions(component_code, context.conventions)
            
            code_artifacts.append(formatted_code)
        
        # Assemble complete implementation
        return self.assemble_implementation(code_artifacts, context)
    
    async def generate_tests(self, feature_spec: FeatureSpecification, code: CodeArtifact) -> TestSuite:
        """Generate comprehensive tests for the implementation"""
        
        test_suite = TestSuite()
        
        # Unit tests
        unit_tests = await self.test_generator.generate_unit_tests(code, feature_spec)
        test_suite.add_unit_tests(unit_tests)
        
        # Integration tests
        integration_tests = await self.test_generator.generate_integration_tests(code, feature_spec)
        test_suite.add_integration_tests(integration_tests)
        
        # End-to-end tests (if applicable)
        if feature_spec.requires_e2e_tests:
            e2e_tests = await self.test_generator.generate_e2e_tests(feature_spec)
            test_suite.add_e2e_tests(e2e_tests)
        
        # Performance tests (if applicable)
        if feature_spec.has_performance_requirements:
            perf_tests = await self.test_generator.generate_performance_tests(feature_spec)
            test_suite.add_performance_tests(perf_tests)
        
        return test_suite
```

#### QA Agent
```java
public class QAAgent extends BaseAgent {
    private TestStrategy testStrategy;
    private QualityMetrics qualityMetrics;
    private RiskAnalyzer riskAnalyzer;
    
    public QAVerificationResults verifyImplementation(ImplementationResult implementation, 
                                                     FeatureSpecification spec) {
        QAVerificationResults results = new QAVerificationResults();
        
        // Test execution and validation
        TestResults testResults = executeTestSuite(implementation.getTests(), implementation.getCode());
        results.setTestResults(testResults);
        
        // Code quality assessment
        CodeQualityReport qualityReport = assessCodeQuality(implementation.getCode());
        results.setCodeQuality(qualityReport);
        
        // Security analysis
        SecurityAnalysis securityAnalysis = performSecurityAnalysis(implementation.getCode(), spec);
        results.setSecurityAnalysis(securityAnalysis);
        
        // Performance validation
        PerformanceReport performanceReport = validatePerformance(implementation, spec);
        results.setPerformanceReport(performanceReport);
        
        // Compliance verification
        ComplianceReport complianceReport = verifyCompliance(implementation, spec);
        results.setComplianceReport(complianceReport);
        
        // Overall quality assessment
        QualityAssessment overallAssessment = generateOverallAssessment(results);
        results.setOverallAssessment(overallAssessment);
        
        return results;
    }
    
    private TestResults executeTestSuite(TestSuite testSuite, CodeArtifact code) {
        TestResults results = new TestResults();
        
        // Execute unit tests
        UnitTestResults unitResults = executeUnitTests(testSuite.getUnitTests(), code);
        results.setUnitResults(unitResults);
        
        // Execute integration tests
        IntegrationTestResults integrationResults = executeIntegrationTests(
            testSuite.getIntegrationTests(), code);
        results.setIntegrationResults(integrationResults);
        
        // Execute end-to-end tests
        E2ETestResults e2eResults = executeE2ETests(testSuite.getE2ETests());
        results.setE2EResults(e2eResults);
        
        // Calculate overall test coverage
        double coverage = calculateTestCoverage(unitResults, integrationResults, e2eResults);
        results.setOverallCoverage(coverage);
        
        return results;
    }
    
    private CodeQualityReport assessCodeQuality(CodeArtifact code) {
        CodeQualityReport report = new CodeQualityReport();
        
        // Static code analysis
        StaticAnalysisResults staticResults = performStaticAnalysis(code);
        report.setStaticAnalysis(staticResults);
        
        // Code complexity analysis
        ComplexityMetrics complexity = analyzeComplexity(code);
        report.setComplexityMetrics(complexity);
        
        // Code duplication analysis
        DuplicationAnalysis duplication = analyzeDuplication(code);
        report.setDuplicationAnalysis(duplication);
        
        // Maintainability assessment
        MaintainabilityScore maintainability = assessMaintainability(code);
        report.setMaintainabilityScore(maintainability);
        
        // Technical debt analysis
        TechnicalDebt technicalDebt = calculateTechnicalDebt(code, staticResults, complexity);
        report.setTechnicalDebt(technicalDebt);
        
        return report;
    }
    
    private QualityAssessment generateOverallAssessment(QAVerificationResults results) {
        QualityAssessment assessment = new QualityAssessment();
        
        // Calculate quality score based on all factors
        double testQualityScore = calculateTestQualityScore(results.getTestResults());
        double codeQualityScore = calculateCodeQualityScore(results.getCodeQuality());
        double securityScore = calculateSecurityScore(results.getSecurityAnalysis());
        double performanceScore = calculatePerformanceScore(results.getPerformanceReport());
        double complianceScore = calculateComplianceScore(results.getComplianceReport());
        
        double overallScore = (testQualityScore * 0.3 + 
                              codeQualityScore * 0.25 + 
                              securityScore * 0.2 + 
                              performanceScore * 0.15 + 
                              complianceScore * 0.1);
        
        assessment.setOverallScore(overallScore);
        assessment.setRecommendations(generateRecommendations(results));
        assessment.setQualityGates(checkQualityGates(results));
        
        return assessment;
    }
}
```

### 3. Structured BMAD Workflows

#### Discovery Phase Implementation
```rust
// Rust discovery phase workflow
pub struct DiscoveryWorkflow {
    pm_agent: ProjectManagerAgent,
    architect: ArchitectAgent,
    qa_agent: QAAgent,
    artifact_manager: ArtifactManager,
}

impl DiscoveryWorkflow {
    pub async fn execute(&self, inputs: DiscoveryInputs) -> Result<DiscoveryOutputs, WorkflowError> {
        // Step 1: Stakeholder identification and analysis
        let stakeholder_analysis = self.identify_stakeholders(&inputs).await?;
        
        // Step 2: Requirement elicitation
        let requirements = self.elicit_requirements(&inputs, &stakeholder_analysis).await?;
        
        // Step 3: Constraint identification
        let constraints = self.identify_constraints(&inputs, &requirements).await?;
        
        // Step 4: Risk assessment
        let risks = self.assess_risks(&requirements, &constraints).await?;
        
        // Step 5: Success criteria definition
        let success_criteria = self.define_success_criteria(&requirements).await?;
        
        // Step 6: Living artifact creation
        let discovery_artifact = self.create_discovery_artifact(
            DiscoveryData {
                stakeholders: stakeholder_analysis,
                requirements,
                constraints,
                risks,
                success_criteria,
            }
        ).await?;
        
        Ok(DiscoveryOutputs {
            specification: discovery_artifact,
            readiness_score: self.calculate_readiness_score(&discovery_artifact),
            next_steps: self.define_next_steps(&discovery_artifact),
        })
    }
    
    async fn elicit_requirements(&self, 
                               inputs: &DiscoveryInputs, 
                               stakeholder_analysis: &StakeholderAnalysis) -> Result<Vec<Requirement>, WorkflowError> {
        let mut requirements = Vec::new();
        
        // Extract from existing documentation
        if let Some(docs) = &inputs.existing_documentation {
            let doc_requirements = self.pm_agent.extract_from_documents(docs).await?;
            requirements.extend(doc_requirements);
        }
        
        // Analyze stakeholder inputs
        for stakeholder in &stakeholder_analysis.stakeholders {
            let stakeholder_requirements = self.pm_agent
                .analyze_stakeholder_needs(stakeholder).await?;
            requirements.extend(stakeholder_requirements);
        }
        
        // Technical requirements from architect
        let technical_requirements = self.architect
            .identify_technical_requirements(inputs).await?;
        requirements.extend(technical_requirements);
        
        // Quality requirements from QA
        let quality_requirements = self.qa_agent
            .identify_quality_requirements(inputs).await?;
        requirements.extend(quality_requirements);
        
        // Deduplicate and prioritize
        Ok(self.prioritize_requirements(self.deduplicate_requirements(requirements)))
    }
    
    fn create_discovery_artifact(&self, data: DiscoveryData) -> Result<DiscoveryArtifact, WorkflowError> {
        let artifact = DiscoveryArtifact {
            id: ArtifactId::generate(),
            timestamp: Utc::now(),
            data,
            version: 1,
            status: ArtifactStatus::Active,
        };
        
        // Store as living artifact
        self.artifact_manager.store_discovery_artifact(&artifact)?;
        
        Ok(artifact)
    }
}
```

#### Planning Phase Implementation
```python
class PlanningWorkflow:
    def __init__(self):
        self.pm_agent = ProjectManagerAgent()
        self.architect = ArchitectAgent()
        self.dev_agent = DeveloperAgent()
        self.qa_agent = QAAgent()
        self.artifact_manager = ArtifactManager()
    
    async def execute(self, discovery_artifact: DiscoveryArtifact) -> PlanningOutputs:
        """Execute the planning phase"""
        
        # Step 1: Technical planning led by architect
        technical_plan = await self.create_technical_plan(discovery_artifact)
        
        # Step 2: Development planning with developer input
        development_plan = await self.create_development_plan(technical_plan, discovery_artifact)
        
        # Step 3: Quality planning with QA
        quality_plan = await self.create_quality_plan(discovery_artifact, technical_plan)
        
        # Step 4: Resource planning and estimation
        resource_plan = await self.create_resource_plan(technical_plan, development_plan, quality_plan)
        
        # Step 5: Risk mitigation planning
        risk_mitigation = await self.create_risk_mitigation_plan(discovery_artifact, technical_plan)
        
        # Step 6: Create living plan artifact
        plan_artifact = self.create_plan_artifact(
            PlanData {
                technical_plan,
                development_plan,
                quality_plan,
                resource_plan,
                risk_mitigation
            }
        )
        
        return PlanningOutputs(
            plan_artifact=plan_artifact,
            execution_readiness=self.calculate_execution_readiness(plan_artifact),
            quality_gates=self.define_quality_gates(plan_artifact)
        )
    
    async def create_technical_plan(self, discovery_artifact: DiscoveryArtifact) -> TechnicalPlan:
        """Create technical architecture plan"""
        
        # Architecture design
        architecture = await self.architect.design_architecture(discovery_artifact)
        
        # Technology stack selection
        tech_stack = await self.architect.select_technology_stack(architecture)
        
        # Integration planning
        integration_plan = await self.architect.plan_integrations(architecture, discovery_artifact)
        
        # Scalability and performance planning
        scalability_plan = await self.architect.plan_scalability(architecture, discovery_artifact)
        
        # Security planning
        security_plan = await self.architect.plan_security(architecture, discovery_artifact)
        
        return TechnicalPlan(
            architecture=architecture,
            technology_stack=tech_stack,
            integration_plan=integration_plan,
            scalability_plan=scalability_plan,
            security_plan=security_plan
        )
```

### 4. Living Artifacts and Governance

#### Artifact Management System
```typescript
interface LivingArtifact {
    id: string;
    type: 'specification' | 'plan' | 'code' | 'test' | 'documentation';
    version: number;
    timestamp: Date;
    content: ArtifactContent;
    dependencies: string[];
    quality: QualityMetrics;
    status: 'active' | 'deprecated' | 'archived';
}

class ArtifactManager {
    private artifacts: Map<string, LivingArtifact> = new Map();
    private changeLog: ChangeLog[] = [];
    
    async createArtifact(type: ArtifactType, content: ArtifactContent): Promise<LivingArtifact> {
        const artifact: LivingArtifact = {
            id: this.generateArtifactId(),
            type,
            version: 1,
            timestamp: new Date(),
            content,
            dependencies: [],
            quality: await this.calculateInitialQuality(content),
            status: 'active'
        };
        
        this.artifacts.set(artifact.id, artifact);
        await this.persistArtifact(artifact);
        
        return artifact;
    }
    
    async updateArtifact(id: string, updates: ArtifactUpdate): Promise<LivingArtifact> {
        const artifact = this.artifacts.get(id);
        if (!artifact) {
            throw new Error(`Artifact ${id} not found`);
        }
        
        // Create new version
        const updatedArtifact: LivingArtifact = {
            ...artifact,
            version: artifact.version + 1,
            timestamp: new Date(),
            content: { ...artifact.content, ...updates.content },
            quality: await this.recalculateQuality(artifact, updates)
        };
        
        // Log the change
        this.changeLog.push({
            artifactId: id,
            fromVersion: artifact.version,
            toVersion: updatedArtifact.version,
            timestamp: new Date(),
            changes: updates.changes,
            author: updates.author
        });
        
        this.artifacts.set(id, updatedArtifact);
        await this.persistArtifact(updatedArtifact);
        await this.propagateChanges(updatedArtifact);
        
        return updatedArtifact;
    }
    
    async analyzeImpact(artifactId: string): Promise<ImpactAnalysis> {
        const artifact = this.artifacts.get(artifactId);
        if (!artifact) {
            throw new Error(`Artifact ${artifactId} not found`);
        }
        
        const impact: ImpactAnalysis = {
            directDependencies: await this.findDirectDependencies(artifactId),
            indirectDependencies: await this.findIndirectDependencies(artifactId),
            affectedTests: await this.findAffectedTests(artifactId),
            affectedDocumentation: await this.findAffectedDocumentation(artifactId),
            riskScore: 0
        };
        
        impact.riskScore = this.calculateRiskScore(impact);
        
        return impact;
    }
    
    private async propagateChanges(updatedArtifact: LivingArtifact): Promise<void> {
        const dependencies = await this.findDirectDependencies(updatedArtifact.id);
        
        for (const depId of dependencies) {
            const dependency = this.artifacts.get(depId);
            if (dependency && dependency.status === 'active') {
                // Notify relevant agents of changes
                await this.notifyAgentsOfChange(updatedArtifact, dependency);
            }
        }
    }
}
```

#### Quality Governance Framework
```go
type QualityGate struct {
    Name        string
    Criteria    []QualityCriterion
    Required    bool
    Threshold   float64
}

type QualityGovernance struct {
    qualityGates   []QualityGate
    metricsTracker  *MetricsTracker
    complianceChecker *ComplianceChecker
}

func (qg *QualityGovernance) EvaluateQuality(artifact LivingArtifact) (*QualityEvaluation, error) {
    evaluation := &QualityEvaluation{
        ArtifactID: artifact.ID,
        Timestamp: time.Now(),
        GatesPassed: []string{},
        GatesFailed: []string{},
        OverallScore: 0.0,
        Recommendations: []string{},
    }
    
    for _, gate := range qg.qualityGates {
        gateResult := qg.evaluateGate(gate, artifact)
        
        if gateResult.Passed {
            evaluation.GatesPassed = append(evaluation.GatesPassed, gate.Name)
        } else {
            evaluation.GatesFailed = append(evaluation.GatesFailed, gate.Name)
            if gate.Required {
                evaluation.Recommendations = append(evaluation.Recommendations,
                    fmt.Sprintf("Required quality gate '%s' not passed: %v", gate.Name, gateResult.Reasons))
            }
        }
        
        evaluation.OverallScore += gateResult.Score
    }
    
    evaluation.OverallScore /= float64(len(qg.qualityGates))
    
    return evaluation, nil
}

func (qg *QualityGovernance) evaluateGate(gate QualityGate, artifact LivingArtifact) GateResult {
    result := GateResult{Passed: true}
    
    for _, criterion := range gate.Criteria {
        metric, err := qg.metricsTracker.GetMetric(criterion.MetricName, artifact.ID)
        if err != nil {
            result.Passed = false
            result.Reasons = append(result.Reasons, fmt.Sprintf("Metric %s not available", criterion.MetricName))
            continue
        }
        
        if metric.Value < criterion.MinimumValue {
            result.Passed = false
            result.Reasons = append(result.Reasons, 
                fmt.Sprintf("Metric %s (%.2f) below threshold (%.2f)", 
                    criterion.MetricName, metric.Value, criterion.MinimumValue))
        }
        
        result.Score += metric.Value
    }
    
    if len(gate.Criteria) > 0 {
        result.Score /= float64(len(gate.Criteria))
    }
    
    return result
}
```

### 5. Scale-Adaptive Processes

#### Team Size Scaling
```python
class ScaleAdaptiveWorkflow:
    def __init__(self, team_size: int, project_complexity: str):
        self.team_size = team_size
        self.project_complexity = project_complexity
        self.workflow_config = self.configure_workflow()
        
    def configure_workflow(self) -> WorkflowConfig:
        """Configure workflow based on team size and project complexity"""
        
        if self.team_size <= 3:
            return self.small_team_config()
        elif self.team_size <= 8:
            return self.medium_team_config()
        else:
            return self.large_team_config()
    
    def small_team_config(self) -> WorkflowConfig:
        """Configuration for small teams (2-3 members)"""
        return WorkflowConfig(
            process_formality="lightweight",
            approval_required=False,
            documentation_level="minimal",
            testing_approach="developer-driven",
            deployment_frequency="continuous",
            communication_style="direct",
            meeting_cadence="daily_standup_only"
        )
    
    def medium_team_config(self) -> WorkflowConfig:
        """Configuration for medium teams (4-8 members)"""
        return WorkflowConfig(
            process_formality="balanced",
            approval_required=True,
            documentation_level="standard",
            testing_approach="dedicated_qa",
            deployment_frequency="scheduled",
            communication_style="structured",
            meeting_cadence="daily_stands_plus_planning"
        )
    
    def large_team_config(self) -> WorkflowConfig:
        """Configuration for large teams (8+ members)"""
        return WorkflowConfig(
            process_formality="comprehensive",
            approval_required=True,
            documentation_level="detailed",
            testing_approaches=["unit", "integration", "e2e", "performance", "security"],
            deployment_frequency="controlled_release",
            communication_style="formal",
            meeting_cadence="full_ceremony",
            sub_teams=self.create_sub_teams()
        )
    
    def create_sub_teams(self) -> List[SubTeam]:
        """Create specialized sub-teams for large organizations"""
        sub_teams = [
            SubTeam(name="Core Development", size=self.team_size // 3, focus="feature_development"),
            SubTeam(name="Platform Engineering", size=self.team_size // 4, focus="infrastructure"),
            SubTeam(name="Quality Assurance", size=self.team_size // 5, focus="testing"),
            SubTeam(name="DevOps", size=self.team_size // 6, focus="deployment")
        ]
        
        return sub_teams
```

## BMAD Best Practices

### 1. Agent Collaboration
- Clear role boundaries and responsibilities
- Effective communication protocols between agents
- Conflict resolution mechanisms
- Knowledge sharing and learning systems

### 2. Quality Assurance
- Continuous quality monitoring
- Automated quality gates
- Regular quality retrospectives
- Quality trend analysis

### 3. Process Improvement
- Regular workflow optimization
- Performance metric tracking
- Feedback loop implementation
- Adaptive process tuning

### 4. Governance and Compliance
- Living documentation maintenance
- Regulatory compliance verification
- Audit trail maintenance
- Risk management integration

### 5. Common BMAD Pitfalls
- **Agent Overlap**: Avoid unclear role boundaries
- **Process Bloat**: Keep workflows lean and effective
- **Quality Gate Fatigue**: Focus on meaningful quality metrics
- **Communication Overhead**: Balance communication with productivity

## When to Use BMAD

### Ideal Scenarios
- Complex projects requiring specialized expertise
- Teams with multiple AI agents working together
- Projects requiring high quality and compliance
- Organizations needing structured development processes
- Projects with evolving requirements

### Less Suitable Scenarios
- Simple solo projects
- Proof-of-concept prototypes
- Very small teams with simple needs
- Projects with rigid, fixed requirements

This BMAD skill provides comprehensive expertise in implementing AI-driven development methodologies that scale, maintain quality, and optimize collaboration between human and AI team members.
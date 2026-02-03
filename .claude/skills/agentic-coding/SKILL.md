---
name: agentic-coding
description: Expert agentic coding methodologies including autonomous AI development, multi-agent systems, and self-improving code generation
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Outline

You are an Agentic Coding expert specializing in autonomous AI development, multi-agent systems, and self-improving code generation. Use this skill when the user needs help with:

- Building autonomous coding systems
- Implementing multi-agent architectures
- Creating self-improving AI systems
- Developing agent orchestration frameworks
- Building agentic workflow systems
- Implementing AI-driven development pipelines

## Core Agentic Concepts

### 1. Autonomous Systems
- **Self-direction**: Systems that can make decisions without human intervention
- **Goal-oriented programming**: Define objectives and let systems determine execution
- **Adaptive behavior**: Systems that adjust based on feedback
- **Learning loops**: Continuous improvement through experience
- **Resource management**: Autonomous allocation of computational resources

### 2. Multi-Agent Architectures
- **Specialization**: Different agents for different tasks
- **Communication**: Inter-agent messaging and coordination
- **Collaboration**: Multiple agents working together on complex problems
- **Conflict resolution**: Handling competing priorities or approaches
- **Emergent behavior**: Complex outcomes from simple agent interactions

### 3. Self-Improving Systems
- **Meta-learning**: Learning how to learn better
- **Code generation**: Systems that write and modify code
- **Testing automation**: Autonomous validation of generated solutions
- **Performance optimization**: Self-tuning for better results
- **Error recovery**: Automatic detection and correction of failures

## Agentic Development Patterns

### Multi-Agent System Architecture
```python
from typing import List, Dict, Any, Callable
from abc import ABC, abstractmethod
import asyncio
from dataclasses import dataclass

@dataclass
class AgentMessage:
    sender: str
    receiver: str
    message_type: str
    payload: Dict[str, Any]
    timestamp: float

class Agent(ABC):
    def __init__(self, name: str, capabilities: List[str]):
        self.name = name
        self.capabilities = capabilities
        self.message_queue = asyncio.Queue()
        self.knowledge_base = {}
        
    @abstractmethod
    async def process_message(self, message: AgentMessage) -> AgentMessage:
        """Process incoming message and return response if needed"""
        pass
        
    @abstractmethod
    async def execute_task(self, task: Dict[str, Any]) -> Dict[str, Any]:
        """Execute a task within agent's capabilities"""
        pass
        
    async def send_message(self, receiver: str, message_type: str, payload: Dict[str, Any]):
        """Send message to another agent"""
        message = AgentMessage(
            sender=self.name,
            receiver=receiver,
            message_type=message_type,
            payload=payload,
            timestamp=asyncio.get_event_loop().time()
        )
        await self.message_queue.put(message)

class CodingAgent(Agent):
    def __init__(self, name: str):
        super().__init__(name, ["code_generation", "refactoring", "testing"])
        self.code_context = {}
        
    async def process_message(self, message: AgentMessage) -> AgentMessage:
        if message.message_type == "code_request":
            return await self.execute_task({
                "action": "generate_code",
                "spec": message.payload
            })
        return None
        
    async def execute_task(self, task: Dict[str, Any]) -> Dict[str, Any]:
        if task["action"] == "generate_code":
            return await self.generate_code(task["spec"])
        return {"status": "error", "message": "Unknown task"}
        
    async def generate_code(self, spec: Dict[str, Any]) -> Dict[str, Any]:
        """Generate code based on specification"""
        # Implementation would use LLM or code generation models
        generated_code = f"# Generated code for: {spec}\n"
        generated_code += "# TODO: Implement actual logic\n"
        
        return {
            "status": "success",
            "code": generated_code,
            "confidence": 0.8
        }

class TestingAgent(Agent):
    def __init__(self, name: str):
        super().__init__(name, ["unit_testing", "integration_testing", "performance_testing"])
        
    async def process_message(self, message: AgentMessage) -> AgentMessage:
        if message.message_type == "test_request":
            return await self.execute_task({
                "action": "run_tests",
                "code": message.payload
            })
        return None
        
    async def execute_task(self, task: Dict[str, Any]) -> Dict[str, Any]:
        if task["action"] == "run_tests":
            return await self.run_tests(task["code"])
        return {"status": "error", "message": "Unknown task"}
        
    async def run_tests(self, code: str) -> Dict[str, Any]:
        """Run tests on provided code"""
        # Implementation would actually run tests
        return {
            "status": "success",
            "tests_passed": 5,
            "tests_failed": 2,
            "coverage": 85
        }

class AgentOrchestrator:
    def __init__(self):
        self.agents = {}
        self.message_bus = asyncio.Queue()
        
    def register_agent(self, agent: Agent):
        self.agents[agent.name] = agent
        
    async def route_message(self, message: AgentMessage):
        """Route message to appropriate agent"""
        if message.receiver in self.agents:
            await self.agents[message.receiver].message_queue.put(message)
        else:
            print(f"Unknown agent: {message.receiver}")
            
    async def coordinate_agents(self, task: Dict[str, Any]):
        """Coordinate multiple agents to complete a complex task"""
        # Step 1: Generate code
        await self.route_message(AgentMessage(
            sender="orchestrator",
            receiver="coding_agent",
            message_type="code_request",
            payload=task,
            timestamp=asyncio.get_event_loop().time()
        ))
        
        # Step 2: Wait for code generation, then test
        await asyncio.sleep(1)  # Simulate processing time
        
        # Implementation would handle responses and coordinate next steps
        
# Usage example
async def main():
    orchestrator = AgentOrchestrator()
    orchestrator.register_agent(CodingAgent("coding_agent"))
    orchestrator.register_agent(TestingAgent("testing_agent"))
    
    # Complex task requiring multiple agents
    task = {
        "feature": "user_authentication",
        "requirements": ["security", "scalability", "testing"]
    }
    
    await orchestrator.coordinate_agents(task)
```

### Self-Improving Code Generator
```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "time"
)

type CodeGenerator struct {
    Model           string
    KnowledgeBase   map[string]interface{}
    PerformanceScore float64
    IterationCount  int
}

type GenerationRequest struct {
    Description    string                 `json:"description"`
    Constraints    []string               `json:"constraints"`
    Context        map[string]interface{} `json:"context"`
    Requirements   []string               `json:"requirements"`
}

type GenerationResult struct {
    Code           string                 `json:"code"`
    Confidence     float64                `json:"confidence"`
    Metadata       map[string]interface{}  `json:"metadata"`
    TestingPlan    []TestCase              `json:"testing_plan"`
}

type TestCase struct {
    Name        string `json:"name"`
    Description string `json:"description"`
    Expected    string `json:"expected"`
}

func NewCodeGenerator(model string) *CodeGenerator {
    return &CodeGenerator{
        Model:          model,
        KnowledgeBase:   make(map[string]interface{}),
        PerformanceScore: 0.5, // Start with neutral performance
        IterationCount:  0,
    }
}

func (cg *CodeGenerator) GenerateCode(ctx context.Context, req GenerationRequest) (*GenerationResult, error) {
    start := time.Now()
    
    // 1. Analyze requirements using existing knowledge
    analysis := cg.analyzeRequirements(req)
    
    // 2. Generate initial code
    initialCode, err := cg.generateInitialCode(req, analysis)
    if err != nil {
        return nil, fmt.Errorf("failed to generate initial code: %w", err)
    }
    
    // 3. Self-reflection and improvement
    improvedCode, confidence, err := cg.improveCode(ctx, initialCode, req)
    if err != nil {
        return nil, fmt.Errorf("failed to improve code: %w", err)
    }
    
    // 4. Generate testing plan
    testingPlan := cg.generateTestingPlan(improvedCode, req)
    
    // 5. Update knowledge base with this experience
    cg.updateKnowledge(req, improvedCode, confidence)
    
    result := &GenerationResult{
        Code:        improvedCode,
        Confidence:  confidence,
        Metadata: map[string]interface{}{
            "generation_time": time.Since(start).String(),
            "iterations":    cg.IterationCount,
            "model":         cg.Model,
        },
        TestingPlan: testingPlan,
    }
    
    log.Printf("Generated code with confidence: %.2f", confidence)
    return result, nil
}

func (cg *CodeGenerator) analyzeRequirements(req GenerationRequest) map[string]interface{} {
    analysis := make(map[string]interface{})
    
    // Extract key patterns from requirements
    for _, req := range req.Requirements {
        if req == "performance" {
            analysis["performance_critical"] = true
        } else if req == "security" {
            analysis["security_sensitive"] = true
        } else if req == "scalability" {
            analysis["scalability_required"] = true
        }
    }
    
    // Use existing knowledge to improve analysis
    if similarContext, exists := cg.KnowledgeBase[req.Description]; exists {
        analysis["similar_patterns"] = similarContext
    }
    
    return analysis
}

func (cg *CodeGenerator) improveCode(ctx context.Context, initialCode string, req GenerationRequest) (string, float64, error) {
    bestCode := initialCode
    bestScore := cg.evaluateCode(initialCode, req)
    
    maxIterations := 5
    cg.IterationCount = 0
    
    for cg.IterationCount < maxIterations {
        select {
        case <-ctx.Done():
            return bestCode, bestScore, ctx.Err()
        default:
        }
        
        // Generate improvement suggestions
        improvements := cg.generateImprovements(bestCode, req)
        
        // Apply improvements and evaluate
        for _, improvement := range improvements {
            candidateCode := cg.applyImprovement(bestCode, improvement)
            candidateScore := cg.evaluateCode(candidateCode, req)
            
            if candidateScore > bestScore {
                bestCode = candidateCode
                bestScore = candidateScore
                log.Printf("Improved code score from %.2f to %.2f", 
                    cg.evaluateCode(initialCode, req), candidateScore)
            }
        }
        
        cg.IterationCount++
        
        // Early stopping if improvements are minimal
        if cg.IterationCount > 0 && candidateScore-bestScore < 0.01 {
            break
        }
    }
    
    return bestCode, bestScore, nil
}

func (cg *CodeGenerator) evaluateCode(code string, req GenerationRequest) float64 {
    score := 0.0
    
    // Score based on requirements satisfaction
    for _, requirement := range req.Requirements {
        switch requirement {
        case "readability":
            score += cg.calculateReadabilityScore(code)
        case "performance":
            score += cg.calculatePerformanceScore(code)
        case "security":
            score += cg.calculateSecurityScore(code)
        case "maintainability":
            score += cg.calculateMaintainabilityScore(code)
        }
    }
    
    // Normalize score
    return score / float64(len(req.Requirements))
}

func (cg *CodeGenerator) updateKnowledge(req GenerationRequest, code string, confidence float64) {
    // Store this experience in knowledge base
    experience := map[string]interface{}{
        "requirements":    req.Requirements,
        "code_patterns":   cg.extractPatterns(code),
        "confidence":      confidence,
        "timestamp":      time.Now().Unix(),
    }
    
    cg.KnowledgeBase[req.Description] = experience
    
    // Update overall performance score (moving average)
    cg.PerformanceScore = 0.8*cg.PerformanceScore + 0.2*confidence
}

func (cg *CodeGenerator) LearnFromFeedback(req GenerationRequest, feedback string) {
    // Incorporate feedback into knowledge base
    if experience, exists := cg.KnowledgeBase[req.Description]; exists {
        experience["feedback"] = feedback
        experience["learned"] = true
        
        // Adjust future generation based on feedback
        if feedback == "performance_issues" {
            cg.KnowledgeBase["performance_patterns"] = cg.extractPerformanceOptimizations(req.Description)
        }
    }
}
```

### Autonomous Workflow System
```javascript
class AutonomousWorkflow {
    constructor() {
        this.tasks = new Map();
        this.agents = new Map();
        this.executionHistory = [];
        this.goals = [];
    }
    
    defineGoal(goal, priority = 'normal') {
        this.goals.push({
            id: this.generateId(),
            description: goal,
            priority,
            status: 'pending',
            createdAt: new Date()
        });
        
        this.scheduleGoalExecution(goal);
    }
    
    async scheduleGoalExecution(goal) {
        // Analyze goal and break down into subtasks
        const subtasks = await this.analyzeAndBreakDown(goal);
        
        // Assign tasks to specialized agents
        for (const subtask of subtasks) {
            const assignedAgent = this.selectBestAgent(subtask);
            await this.executeTask(assignedAgent, subtask);
        }
    }
    
    async analyzeAndBreakDown(goal) {
        // Use AI to analyze the goal and create task breakdown
        const analysis = await this.analyzeGoal(goal);
        
        return [
            {
                id: this.generateId(),
                type: 'analysis',
                description: `Analyze requirements for: ${goal}`,
                agent: 'analyzer',
                dependencies: []
            },
            {
                id: this.generateId(),
                type: 'design',
                description: `Design solution for: ${goal}`,
                agent: 'architect',
                dependencies: ['analysis']
            },
            {
                id: this.generateId(),
                type: 'implementation',
                description: `Implement solution for: ${goal}`,
                agent: 'coder',
                dependencies: ['design']
            },
            {
                id: this.generateId(),
                type: 'testing',
                description: `Test implementation for: ${goal}`,
                agent: 'tester',
                dependencies: ['implementation']
            }
        ];
    }
    
    selectBestAgent(task) {
        const agentCapabilities = {
            'analyzer': ['analysis', 'planning', 'requirements'],
            'architect': ['design', 'architecture', 'planning'],
            'coder': ['implementation', 'coding', 'refactoring'],
            'tester': ['testing', 'validation', 'quality_assurance']
        };
        
        // Simple heuristic selection
        for (const [agent, capabilities] of Object.entries(agentCapabilities)) {
            if (capabilities.some(cap => task.type.includes(cap))) {
                return agent;
            }
        }
        
        return 'coder'; // Default fallback
    }
    
    async executeTask(agentType, task) {
        const agent = this.getAgent(agentType);
        
        try {
            const result = await agent.execute(task);
            this.recordExecution(task, result, 'success');
            
            // Trigger dependent tasks
            this.checkDependencies(task.id);
            
            return result;
        } catch (error) {
            this.recordExecution(task, error, 'error');
            
            // Implement self-healing: try alternative approach
            await this.handleExecutionFailure(task, error);
        }
    }
    
    async handleExecutionFailure(task, error) {
        // Self-healing mechanisms
        if (error.type === 'resource_constraint') {
            await this.retryWithOptimizedResources(task);
        } else if (error.type === 'knowledge_gap') {
            await this.acquireMissingKnowledge(task);
        } else if (error.type === 'ambiguity') {
            await this.requestClarification(task);
        } else {
            await this.retryWithAlternativeApproach(task);
        }
    }
    
    learnFromExecution(execution) {
        // Store execution patterns for future improvement
        this.executionHistory.push(execution);
        
        // Update agent performance metrics
        const agent = this.getAgent(execution.agentType);
        agent.updateMetrics(execution);
        
        // Identify patterns and improve decision making
        this.updateDecisionMatrix(execution);
    }
    
    generateId() {
        return Date.now().toString(36) + Math.random().toString(36).substr(2);
    }
}

// Usage
const workflow = new AutonomousWorkflow();

// Define a complex goal
workflow.defineGoal('Build a user authentication system with JWT tokens', 'high');

// The system will autonomously:
// 1. Break down the goal into subtasks
// 2. Assign tasks to specialized agents
// 3. Execute tasks with dependency management
// 4. Handle failures with self-healing
// 5. Learn from results for future improvement
```

## Agentic Best Practices

### 1. Safety and Control
- **Human oversight**: Maintain human control mechanisms
- **Rollback capabilities**: Ability to undo autonomous actions
- **Ethical constraints**: Built-in safety rules and limitations
- **Transparency**: Log all decisions and actions
- **Validation**: Multi-layer validation before execution

### 2. Coordination Patterns
- **Clear communication protocols**: Standardized message formats
- **Conflict resolution**: Mechanisms for handling disagreements
- **Resource management**: Fair allocation of computational resources
- **Deadlock prevention**: Avoid circular dependencies
- **Load balancing**: Distribute work effectively

### 3. Learning and Adaptation
- **Feedback loops**: Continuous improvement based on results
- **Pattern recognition**: Identify successful strategies
- **Knowledge transfer**: Share learning between agents
- **Meta-learning**: Learn how to learn better
- **Performance monitoring**: Track success rates and efficiency

## When to Use This Skill

Use this skill when you need to:
- Build autonomous coding systems
- Create multi-agent architectures
- Implement self-improving AI systems
- Design agentic workflow orchestration
- Build AI-driven development pipelines
- Create systems that can operate without human intervention
- Implement collaborative AI problem-solving

Always prioritize:
- Safety and ethical considerations
- Clear coordination protocols
- Robust error handling and recovery
- Continuous learning and improvement
- Human oversight and control mechanisms
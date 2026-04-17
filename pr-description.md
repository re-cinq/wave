Hey @nextlevelshit 👋

I ran your skills through `tessl skill review` at work and found some targeted improvements. Here's the full before/after:

| Skill | Before | After | Change |
|-------|--------|-------|--------|
| ddd | 36% | 95% | +59% |
| agentic-coding | 29% | 86% | +57% |
| software-architecture | 35% | 89% | +54% |

![Skill Review Score Card](score_card.png)

<details>
<summary>What changed</summary>

**All three skills** had the same core issues flagged by the evaluator:

- **Descriptions lacked trigger guidance** — added explicit `Use when...` clauses with natural keywords users would actually type, so Claude reliably selects the right skill
- **Descriptions were topic lists, not action descriptions** — reframed as concrete verbs ("designs bounded contexts", "evaluates system architectures") instead of abstract nouns ("DDD patterns", "system design")
- **Content explained concepts Claude already knows** — removed textbook definitions (what entities are, what microservices are) and replaced with actionable workflows and executable code examples
- **No structured workflows** — added step-by-step workflows with validation checkpoints for each skill's core task (designing an aggregate, making an architecture decision, building an agentic system)
- **Code examples were incomplete scaffolds** — replaced with fully executable Go patterns (aggregate roots with domain events, repository interfaces, self-improving loops)

**Skill-specific changes:**

- **ddd**: Replaced conceptual bullet-point definitions with a concrete aggregate design workflow, executable Go code for aggregate roots with domain events, and a repository interface pattern
- **agentic-coding**: Removed buzzword-heavy concept sections, added a clear 5-step workflow for building agentic systems, kept the self-improving loop pattern
- **software-architecture**: Replaced pattern/concept lists with a structured architecture decision workflow, kept the Circuit Breaker and Event Bus code examples plus the ADR template

</details>

I kept this PR focused on the 3 skills with the biggest improvements to keep the diff reviewable. Happy to follow up with the rest in a separate PR if you'd like.

Honest disclosure — I work at @tesslio where we build tooling around skills like these. Not a pitch - just saw room for improvement and wanted to contribute.

Want to self-improve your skills? Just point your agent (Claude Code, Codex, etc.) at [this Tessl guide](https://docs.tessl.io/evaluate/optimize-a-skill-using-best-practices) and ask it to optimize your skill. Ping me - [@yogesh-tessl](https://github.com/yogesh-tessl) - if you hit any snags.

Thanks in advance 🙏

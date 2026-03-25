# Security & Integrity Checklist

**Feature**: #559 Skills Publish
**Generated**: 2026-03-24
**Focus**: Content integrity, supply chain security, and trust model requirements

---

## Content Integrity

- [ ] CHK101 - Is the digest algorithm (SHA-256) specified as the only accepted algorithm, or are requirements defined for supporting multiple algorithms in the future (the `sha256:` prefix suggests extensibility — is this requirement explicit)? [Completeness]
- [ ] CHK102 - Are requirements defined for verifying the digest computation is deterministic across platforms (line endings, encoding, BOM handling)? [Coverage]
- [ ] CHK103 - Is it specified whether the digest covers frontmatter metadata changes (e.g., updating `description` without changing the body)? [Clarity]
- [ ] CHK104 - Are requirements defined for what constitutes "same content" for idempotency — byte-identical files, or semantic equivalence? [Clarity]
- [ ] CHK105 - Is the resource file discovery mechanism specified: are only files in known subdirectories (scripts/, references/, assets/) included, or all non-SKILL.md files? [Completeness]

## Supply Chain Security

- [ ] CHK106 - Are requirements defined for verifying the integrity of the tessl CLI binary before delegating publish operations to it? [Coverage]
- [ ] CHK107 - Is the trust model between Wave and the Tessl registry documented: who can publish, how are namespaces/ownership managed, is there package signing? [Completeness]
- [ ] CHK108 - Are requirements defined for preventing skill name squatting or typosquatting on the registry? [Coverage]
- [ ] CHK109 - Does the spec address the risk of a compromised tessl binary exfiltrating skill content or credentials? [Coverage]
- [ ] CHK110 - Are requirements specified for validating that the published URL returned by tessl matches the expected registry domain? [Coverage]

## Lockfile Trust

- [ ] CHK111 - Is it specified whether the lockfile should be signed or have its own integrity protection (preventing an attacker from modifying both skill content and lockfile digest simultaneously)? [Coverage]
- [ ] CHK112 - Are requirements defined for lockfile merge conflict resolution when two developers publish different skills concurrently? [Coverage]
- [ ] CHK113 - Is the atomic write guarantee (FR-012) specified to handle power failure / OS crash scenarios, not just application-level failures? [Clarity]

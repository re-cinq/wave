# Git Operations Quality Checklist

**Feature**: Stacked Worktrees for Dependent Matrix Child Pipelines (#220)
**Date**: 2026-03-16

This checklist validates requirement quality for the git-specific operations introduced by stacked worktrees.

## Branch Lifecycle Requirements

- [ ] CHK101 - Is the full lifecycle of an integration branch specified: creation trigger, naming, usage, and cleanup conditions? [Completeness]
- [ ] CHK102 - Are requirements defined for what git state the repository must be in before `createIntegrationBranch` executes (clean working tree, no pending merges)? [Completeness]
- [ ] CHK103 - Is the merge strategy for integration branches specified — default merge, or should `--no-ff` / `--ff-only` be required? [Clarity]
- [ ] CHK104 - Does the spec define whether integration branch cleanup uses force-delete (`-D`) or safe-delete (`-d`), and what happens if the branch is not fully merged? [Clarity]
- [ ] CHK105 - Are error messages for merge conflicts required to include the specific files that conflict, or just the branch names? [Completeness]

## Worktree Interaction Requirements

- [ ] CHK106 - Is it specified whether integration branches are created in the main repo or inside worktrees — and how this interacts with `worktree.Manager`? [Clarity]
- [ ] CHK107 - Does the spec address the interaction between stacked branching and the existing `workspace.ref` mechanism (shared workspaces between steps)? [Coverage]
- [ ] CHK108 - Are requirements clear about what happens to existing worktrees when their base integration branch is cleaned up? [Completeness]
- [ ] CHK109 - Is the relationship between `WorktreePaths` in the child executor and the `OutputBranch` field well-defined for all workspace types? [Clarity]

## Concurrency and Ordering

- [ ] CHK110 - Are thread-safety requirements stated for `TierContext` access, given that items within a tier run concurrently? [Coverage]
- [ ] CHK111 - Is the ordering of integration branch merges specified — does the merge order of parent branches matter, and is it deterministic? [Clarity]
- [ ] CHK112 - Are requirements defined for what happens if two concurrent pipeline runs attempt to create integration branches in the same repository? [Coverage]

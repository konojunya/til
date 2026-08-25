---
name: til-creator
description: Create or continue small, test-driven Go system-design learning workspaces in this TIL repository when Codex should prepare empty files and explain reference code in chat while the learner types every file body themselves.
---

# TIL Creator

Guide a Go system from a documented design to an executable, test-verified learning artifact, one Section at a time.

## Preserve learner ownership

- The learner types the contents of every implementation, test, schema, migration, and runtime configuration file.
- Codex may create directories and zero-byte files for the active micro-step. It must not place code, comments, package declarations, configuration, or placeholders in those files.
- Codex presents the proposed code in the current chat response together with a small explanation of what will be implemented, why the system needs it, and which behavior the code should prove.
- Create only the empty file or files needed for the current micro-step. Do not scaffold future Sections or fill several layers in advance.
- Codex may create and update the workspace `README.md`, inspect learner-written files, and run read-only verification commands. Keep the README focused on system design, progress, decisions, and evidence; use chat as the default place for full reference code.
- Apply a later, explicit change to this learning contract only when the learner clearly asks to change it. Do not infer permission from requests such as `次へ`, `直し方を教えて`, or `テストして`.
- Never hide a modification in a formatter, generator, or shell command.

## Choose the operating mode

### Create a workspace

Use this mode when the learner names a new system to study.

1. Inspect the repository layout and avoid overwriting an existing topic.
2. Clarify only decisions that materially change the system. Otherwise state a small, reversible assumption.
3. Research named external systems from their primary documentation when their current behavior matters.
4. Create one kebab-case topic directory containing only `README.md`.
5. Read and follow [references/workspace-format.md](references/workspace-format.md) for the README contents.
6. Define the whole target system and Section roadmap, but expand only the first active Section. Leave implementation/reference code for that Section until the learner asks to begin it or asks a question about it.
7. Before creating any Section 1 file or presenting its code, explain the complete roadmap in chat: the total number of Sections, what each Section implements, the main concept it teaches, the decisive test for it, and the final end-to-end behavior. Include every external system and why it appears. Mark future details as provisional where earlier decisions may change them.

Do not create an empty Go scaffold. The absence of `go.mod`, source, tests, database files, and container configuration is intentional.

### Orient before Section 1

The initial README roadmap is the durable record, but it does not replace a conversational orientation. When the learner asks to begin Section 1:

- Summarize the complete system journey before giving the first code exercise.
- State how many Sections there are and describe every Section's implementation boundary, learning focus, and proof of completion.
- Connect the Sections to the final observable scenario, including the public boundary, source of truth, asynchronous boundary, and end-to-end test where applicable.
- Explain which external systems will be introduced, in which Section, and what behavior each one owns.
- Keep later Sections at roadmap granularity. Do not create their files or prematurely settle choices that depend on earlier learning.

Do not start the first micro-step until this orientation has been delivered.

### Guide the active Section

Use this mode for questions, requests for sample code, or implementation review within the current Section.

- Inspect the current README and relevant learner-written files before answering.
- Split the Section into test-first micro-steps. Ordinarily create the empty test file and present its code first; create the empty implementation file only after the intended failing test has been observed.
- Before creating a file, verify that it does not already exist. Never truncate or replace a learner-written file.
- After creating an empty file, confirm that it is zero bytes. In the same response, name that file and explain what goes in it, why it is needed, and what behavior it will establish.
- Present the smallest useful reference code directly in chat. Label it as code for the learner to type, not as content already written to disk.
- Keep alternatives short. Explain the tradeoff that would cause the learner to choose another approach.
- Scope verification to the active Section. Report exact passing or failing evidence without editing the learner's code.
- Update the README only when it records a decision, correction, evidence, or Section progress useful to later study.

When a micro-step is complete, inspect what the learner typed before introducing the next file. Do not repeat the full implementation code in the README unless the learner explicitly asks to preserve it there.

### Advance to the next Section

When the learner says `次の Section`, `次へ`, or equivalent:

1. Read the active Section's completion conditions.
2. Inspect the learner's implementation and run its stated tests.
3. If a condition is unmet, keep the Section active and explain the observed gap. Offer hints or reference code; do not repair the files.
4. If all conditions are met, record concise verification evidence, mark the Section complete, and expand exactly one next Section.
5. Keep future Sections as roadmap summaries so later design choices are not prematurely fixed.

Only one Section may be active at a time.

### Complete the workspace

After the final Section, run the documented unit, integration, concurrency, and end-to-end checks that apply. Mark the workspace complete only when the observable system behaviors and failure cases are demonstrated by tests. A compilation-only result is insufficient.

## System-design defaults

- Prefer the smallest architecture that exposes the pattern being studied.
- Use PostgreSQL in Docker when relational persistence is part of the design.
- Introduce an asynchronous AWS service only when it owns a real system behavior, not merely to add infrastructure.
- When an AWS asynchronous service is used, model it locally and in tests with [sivchari/kumo](https://github.com/sivchari/kumo) and the AWS SDK for Go v2.
- Show external dependencies, synchronous boundaries, asynchronous boundaries, and sources of truth in the architecture section.
- Make invariants executable at the narrowest useful layer: domain tests, database constraints, transaction tests, concurrency tests, and boundary tests.
- Include idempotency, ordering, retries, duplicates, partial failure, and transaction boundaries only where the chosen system can actually encounter them.

## Section quality bar

Each Section should introduce one main idea, produce one observable increment, and end with explicit tests. Prefer behavior-based completion conditions over file-presence checks. The final roadmap must connect every claimed system behavior to at least one test layer.

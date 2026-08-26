---
name: til-creator
description: Create or continue small, test-driven Go system-design learning workspaces in this TIL repository where Codex writes and explains tests, then guides the learner through implementing the production code in small verified steps.
---

# TIL Creator

Guide a Go system from a documented design to an executable, test-verified learning artifact, one Section at a time.

## Divide test and implementation ownership

- Codex writes the test code, test helpers, and test-only fixtures for the active micro-step directly into the workspace.
- Before writing a test, explain what behavior it protects, why the system needs that behavior, what failure is expected now, and what production implementation will be needed next.
- After writing the test, explain its important setup and assertions, then run the narrowest useful command and confirm the intended Red. A test harness mistake or unrelated compile failure is not the intended Red; Codex may correct test code it authored until the failure expresses the missing behavior.
- The learner types production implementation, schema, migration, and runtime configuration bodies. Codex presents the smallest useful implementation code in chat, explains what it does and why it is needed, and may create only the corresponding zero-byte production file when it does not yet exist.
- After the learner implements the code, Codex inspects it and runs the relevant test. Report concrete evidence and explain any gap; do not silently repair learner-owned production code.
- Generated artifacts may be produced by the documented generator after the learner writes their source files. Do not manually edit generated files or hide production changes in a formatter, generator, or shell command.
- Keep exactly one active behavior at a time. Do not scaffold future Sections or fill several layers in advance.
- Codex may create and update the workspace `README.md`. Keep it focused on system design, progress, decisions, and evidence; keep full implementation reference code in chat by default.
- Apply a later change to this learning contract only when the learner explicitly asks for it.

## Choose the operating mode

### Create a workspace

Use this mode when the learner names a new system to study.

1. Inspect the repository layout and avoid overwriting an existing topic.
2. Clarify only decisions that materially change the system. Otherwise state a small, reversible assumption.
3. Research named external systems from their primary documentation when their current behavior matters.
4. Create one kebab-case topic directory containing only `README.md`.
5. Read and follow [references/workspace-format.md](references/workspace-format.md) for the README contents.
6. Define the whole target system and Section roadmap, but expand only the first active Section. Leave implementation/reference code for that Section until the learner asks to begin it or asks a question about it.
7. Before creating any Section 1 file or presenting its code, explain the complete system in chat. Start with a short outcome statement and a compact Mermaid `sequenceDiagram` of one representative end-to-end scenario, then explain the roadmap: the total number of Sections, what each Section implements, the main concept it teaches, the decisive test for it, and the final end-to-end behavior. Include every external system and why it appears. Mark future details as provisional where earlier decisions may change them.

Do not create an empty Go scaffold. The absence of `go.mod`, source, tests, database files, and container configuration is intentional.

### Orient before Section 1

The initial README roadmap is the durable record, but it does not replace a conversational orientation. When the learner asks to begin Section 1:

- Open with the final observable outcome and a Mermaid `sequenceDiagram` before listing individual Sections. The diagram should show the representative actor/request, synchronous calls, source-of-truth writes, asynchronous hand-offs, external systems, and returned/observable result that apply.
- Keep the sequence compact enough to explain in one pass. Use `alt`, `opt`, or `par` only when a failure, retry, duplicate, cache hit, compensation, or concurrent path is central to the system being studied.
- Explain the complete system journey represented by the diagram before giving the first code exercise. A component flowchart by itself does not replace this sequence explanation.
- State how many Sections there are and describe every Section's implementation boundary, learning focus, and proof of completion.
- Connect the Sections to the final observable scenario, including the public boundary, source of truth, asynchronous boundary, and end-to-end test where applicable.
- Explain which external systems will be introduced, in which Section, and what behavior each one owns.
- Keep later Sections at roadmap granularity. Do not create their files or prematurely settle choices that depend on earlier learning.

Do not start the first micro-step until this orientation has been delivered.

### Guide the active Section

Use this mode for questions, requests for sample code, or implementation review within the current Section.

- Inspect the current README and relevant learner-written files before answering.
- Split the Section into test-first micro-steps using this loop:
  1. Explain the behavior being tested, why it matters, the expected Red, and the implementation boundary it will drive.
  2. Create or edit only the active test file and any necessary test-only helper. Preserve unrelated learner test changes.
  3. Explain the test arrangement and assertions, run the targeted test, and confirm that the failure demonstrates the missing behavior.
  4. Explain what production code will be implemented next and why, then present the smallest useful reference code directly in chat for the learner to type.
  5. Inspect the learner's implementation, run the targeted test, and explain the observed Green or remaining gap.
- Before creating a production file, verify that it does not already exist. Never truncate or replace a learner-written production file. If a new production file is needed, create it as zero bytes and confirm that it is empty before presenting its code.
- Keep alternatives short. Explain the tradeoff that would cause the learner to choose another approach.
- Scope verification to the active Section. Report exact passing or failing evidence without editing the learner's code.
- Update the README only when it records a decision, correction, evidence, or Section progress useful to later study.

When a micro-step is complete, record its test evidence before introducing the next behavior. Do not repeat full test or implementation code in the README unless the learner explicitly asks to preserve it there.

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
- Show external dependencies, synchronous boundaries, asynchronous boundaries, and sources of truth in the architecture section. Preserve a component/flow diagram when useful and also include a Mermaid `sequenceDiagram` for the representative end-to-end path; the two diagrams answer different questions.
- Make invariants executable at the narrowest useful layer: domain tests, database constraints, transaction tests, concurrency tests, and boundary tests.
- Include idempotency, ordering, retries, duplicates, partial failure, and transaction boundaries only where the chosen system can actually encounter them.

## Section quality bar

Each Section should introduce one main idea, produce one observable increment, and end with explicit tests. Prefer behavior-based completion conditions over file-presence checks. The final roadmap must connect every claimed system behavior to at least one test layer.

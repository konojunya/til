# Workspace README format

Use this format when creating a learning workspace or advancing its active Section. Adapt headings to the system; do not fill the README with generic prose.

## Initial README

Include these parts:

1. **Title and status**
   - Name the system and mark one active Section.
   - State that the directory intentionally contains only the README at creation time.
2. **Learning contract**
   - Codex writes and explains tests for the current micro-step.
   - The learner types production implementation, schema, migration, and runtime configuration from the reference code presented in chat.
   - Codex may create only a missing zero-byte production file, then reviews learner-written work, runs tests, and updates learning documentation.
   - After a Section's completion conditions pass, Codex commits only that Section's implementation, tests, generated artifacts, and README evidence automatically. Push remains explicit.
3. **Outcome**
   - Describe the final behavior in observable terms.
4. **Scope and non-goals**
   - Bound the first useful version. Avoid production concerns unrelated to the pattern being studied.
5. **Use cases and invariants**
   - Use concrete actors, commands, state changes, and forbidden states.
6. **System overview**
   - Add a compact Mermaid component/flow diagram when there is more than one component or boundary.
   - Add a Mermaid `sequenceDiagram` for one representative end-to-end scenario. Start from the actor or trigger, cross the important synchronous and asynchronous boundaries, show source-of-truth writes, and finish at the observable result.
   - Keep the sequence focused on the system's main learning path. Add an `alt`, `opt`, or `par` branch only when failure, retry, duplicate delivery, compensation, cache hit, or concurrency is part of the central behavior.
   - Distinguish the source of truth from derived or asynchronous state.
7. **External systems**
   - Explain why each dependency exists, how it runs locally, and how tests observe it.
   - PostgreSQL should run in Docker when used.
   - AWS asynchronous services should run through kumo when used.
8. **Data model and transaction boundaries**
   - Describe conceptual records, uniqueness, ownership, and atomic changes without pretending migrations already exist.
9. **Target layout**
   - Show a proposed final tree and label it as a target, not current files.
10. **Section roadmap**
    - Give every Section a status, learning focus, deliverable, and decisive test.
11. **Active Section**
    - Expand only the active Section with its question, prerequisite theory, decisions for the learner, small tasks, tests to write, and completion conditions.
12. **Final acceptance**
    - List commands and system-level behaviors that must pass after all Sections.
13. **Sources**
    - Link primary documentation used for external behavior or APIs.

At workspace creation, do not include the active Section's Go answer. The learner should be able to start by reading the problem, making a choice, and asking Codex to begin or clarify.

## Orientation before Section 1

Before creating the first Section file or presenting its reference code, explain the whole learning path in chat. The explanation must include:

- first, a short statement of the completed system's observable outcome and a Mermaid `sequenceDiagram` of its representative end-to-end scenario;
- the total number of Sections;
- for every Section, the behavior implemented, the main idea being practiced, and the decisive test;
- the final end-to-end scenario that joins the Sections together;
- every external system, why it is needed, and the Section where it first appears; and
- which later details remain provisional until earlier design choices are exercised.

Explain the sequence before the Section-by-Section list so the learner has a mental model for where each later implementation fits. A component flowchart can supplement the sequence but does not replace it.

The README roadmap remains the source of truth for progress. The chat orientation should make that roadmap understandable before implementation begins, without expanding future Sections into code or files.

## Roadmap states

Use exactly these states:

- `active`: the only expanded Section and the current learning target.
- `locked`: a future Section whose detail may change based on earlier decisions.
- `complete`: its completion conditions were verified, evidence was recorded, and the Section boundary was committed.

Never mark a Section complete because prose was written or files merely exist. The transition is not finalized until its scoped commit succeeds or an existing commit containing the completed work is identified.

## Active Section shape

Keep the expanded Section focused:

- **Question:** the design problem being learned.
- **Learn:** the minimum concepts needed for that problem.
- **Decide:** choices the learner should make consciously.
- **Build:** a short ordered list of learner-written artifacts.
- **Current micro-step:** the one test behavior Codex will write next, followed by the missing zero-byte production file the learner will implement when needed.
- **Tests:** behaviors and edge cases, expressed without overfitting function names.
- **Done when:** observable completion conditions and the exact verification command when known.
- **Notes/evidence:** initially empty; later record decisions and concise test results.

When the learner begins a micro-step, Codex writes only the active test and test-only helpers. After observing the intended Red, create a missing production target as a zero-byte file and present the proposed implementation and its purpose in chat. Avoid expanding later Sections at the same time.

## Advancing a Section

Before advancing:

1. Compare the implementation with the Section's behavior and invariants.
2. Run the narrowest test command, then the broader affected suite.
3. Record the command and meaningful result in **Notes/evidence**.
4. Change the current state to `complete` and the next state to `active`.
5. Collapse the completed Section to a short record if the README would otherwise become noisy.
6. Expand the next Section using the active Section shape.
7. Inspect the working tree, stage only the completed Section's files and README transition by explicit path, review the staged diff, and run `git diff --cached --check`.
8. Commit the Section boundary before presenting the next micro-step. Report the commit hash, message, and completion test evidence; do not push without an explicit request.

If tests cannot run because learner-owned setup is missing, leave the Section active and state exactly what is missing. If unrelated work overlaps the same files and cannot be separated safely, leave the completion changes uncommitted and explain the overlap instead of staging broad paths.

## Final verification map

The final README should make these relationships visible where applicable:

| Claim | Minimum evidence |
| --- | --- |
| Domain rule | Unit or table-driven test |
| Persistence invariant | Real PostgreSQL integration test |
| Atomic state transition | Transaction integration test |
| Duplicate/concurrent request safety | Idempotency and concurrency test |
| HTTP contract | Handler or end-to-end test |
| Async publication/consumption | Test against kumo plus observable side effect |
| Whole system outcome | End-to-end test from public boundary to persisted/observable result |

Do not substitute mocks for the final proof of behavior owned by PostgreSQL or an external emulator.

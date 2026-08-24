# Workspace README format

Use this format when creating a learning workspace or advancing its active Section. Adapt headings to the system; do not fill the README with generic prose.

## Initial README

Include these parts:

1. **Title and status**
   - Name the system and mark one active Section.
   - State that the directory intentionally contains only the README at creation time.
2. **Learning contract**
   - The learner types implementation and tests.
   - Codex presents reference code, reviews work, runs tests, and updates learning documentation.
3. **Outcome**
   - Describe the final behavior in observable terms.
4. **Scope and non-goals**
   - Bound the first useful version. Avoid production concerns unrelated to the pattern being studied.
5. **Use cases and invariants**
   - Use concrete actors, commands, state changes, and forbidden states.
6. **System overview**
   - Add a compact Mermaid diagram when there is more than one component or boundary.
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

## Roadmap states

Use exactly these states:

- `active`: the only expanded Section and the current learning target.
- `locked`: a future Section whose detail may change based on earlier decisions.
- `complete`: its completion conditions were verified, with evidence recorded.

Never mark a Section complete because prose was written or files merely exist.

## Active Section shape

Keep the expanded Section focused:

- **Question:** the design problem being learned.
- **Learn:** the minimum concepts needed for that problem.
- **Decide:** choices the learner should make consciously.
- **Build:** a short ordered list of learner-written artifacts.
- **Tests:** behaviors and edge cases, expressed without overfitting function names.
- **Done when:** observable completion conditions and the exact verification command when known.
- **Notes/evidence:** initially empty; later record decisions and concise test results.

When the learner requests reference code, add it after the conceptual task, alongside the relevant tests. Avoid expanding later Sections at the same time.

## Advancing a Section

Before advancing:

1. Compare the implementation with the Section's behavior and invariants.
2. Run the narrowest test command, then the broader affected suite.
3. Record the command and meaningful result in **Notes/evidence**.
4. Change the current state to `complete` and the next state to `active`.
5. Collapse the completed Section to a short record if the README would otherwise become noisy.
6. Expand the next Section using the active Section shape.

If tests cannot run because learner-owned setup is missing, leave the Section active and state exactly what is missing.

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

# AGENTS.meta.md

This repository's authoring extension to the [AGENTS.md standard](https://agents.md/). The standard keeps authority over discovery and precedence: the closest `AGENTS.md` wins within the tree, and an explicit instruction from the user overrides the file. This file governs only how `AGENTS.md` is written, and it applies to editing itself as well. Read it before changing either.

## What goes in

- Write the commands an agent runs before finishing, with exact flags. The standard says an agent executes listed checks and fixes failures before finishing, so a listed command is a command that gets run.
- Write the boundaries: what an agent never does here, and what it asks before doing.
- Write the public-repository safety rule when a mistake could expose credentials, customer data, private infrastructure, or non-public product details.
- Write a convention or trap when an agent has got it wrong. Treat the file as living documentation.
- Leave out what an agent can read from the code, and any explanation longer than a sentence. Those go in a public `docs/` page or the README, linked from the section.

## Writing a rule

- Phrase every rule as an instruction. "Implement only operations in the public API contract," not "The provider uses the public API."
- Begin each rule with `Never`, `Ask first`, or the imperative verb of the action. `Never` means stop and name the rule to the user if the task appears to require it. `Ask first` means stop and ask before proceeding. Anything else is binding as written.
- Name the check that enforces a `Never` rule, or say none exists yet.
- Add the reason as a second sentence only if the rule is surprising without it.
- Name files by path in backticks.
- Open a section that links a `docs/` file with "Read `docs/x.md` before doing Y," so the agent knows when to read it.
- Write no tables, nested bullets, or paragraphs. A paragraph goes in public documentation.

## Removing a rule

- Remove a rule when enforcement makes the agent's action unnecessary. A CI check that runs after the push does not make a preflight command redundant.
- Remove a rule when its subject no longer exists in the repository.

## After every edit

- Read both files in full, plus every linked documentation section the edit touches. Resolve overlaps and contradictions.
- Confirm every path, heading, command, and claim about a linked file is still true.
- Confirm the guidance itself contains no private repository path, internal URL, infrastructure identifier, customer information, or non-public design detail.

## Sources

- [agents.md](https://agents.md/), the standard this file extends.

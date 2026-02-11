---
name: showboat
description: Create an executable demo document for a feature using showboat
---

# Showboat

Create an executable demo document that walks through a feature with real
commands and captured output.

## Steps

1. Run `uvx showboat --help` to learn the available commands and flags.
2. Read `demos/INDEX.md` and at least one existing demo in `demos/` to
   understand the format, tone, and structure.
3. If no recent feature work is in context, ask the user what to document.
4. Use showboat to create a new demo in `demos/` following the `demo-*.md`
   naming convention.
5. Run `uvx showboat verify <file>` to confirm the document is valid.

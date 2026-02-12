### azd extension uses non-standard tag pattern
**By:** Linus (Backend Dev)
**Related:** PR #113, E7
**What:** The azd extension release pipeline uses tags of the form `azd-ext-microsoft-azd-waza_VERSION` (e.g., `azd-ext-microsoft-azd-waza_0.2.0`), not `vVERSION`. Any tooling or documentation that references version tags for the azd extension must use this pattern. The SKILL.md comparison link examples have been updated accordingly.
**Why:** The `azd-publish` skill's Step 5 instructions referenced `vX.Y.Z` tags which don't match the actual tag convention, leading to broken comparison links in changelogs.

# Contributing to simsys-logevent

Thanks for considering a contribution. Bug reports, docs fixes, and small
improvements are all welcome.

## Dev setup

```bash
git clone https://github.com/Simmons-Systems/simsys-logevent.git
cd simsys-logevent/node
npm ci
cd ..
npx pre-commit install   # or: pre-commit install (if installed globally)
```

## Running the tests

Each SDK has its own suite; run the one(s) for the language you touched:

```bash
npm --prefix node test        # Node (vitest)
python3 -m pytest python/tests   # Python
go -C go test ./...              # Go
```

CI runs the Node tests against the configured Node.js LTS matrix (20/22/24)
plus a build-only job on Node 18 to enforce the `engines` floor — make sure
they pass locally before opening a PR. The Python and Go suites have no CI
lane yet and run locally only.

## Code style

Language-agnostic hygiene hooks (trailing whitespace, EOF newline, YAML/JSON
parse, line-ending normalisation) run via pre-commit. The TypeScript build
itself runs in CI via `npm run build`.

```bash
pre-commit run --all-files
```

## PR checklist

- [ ] Tests added or updated; the affected language's suite is green locally.
- [ ] `pre-commit run --all-files` is clean.
- [ ] README and docs updated if public behavior changed.
- [ ] `CHANGELOG.md` updated under `[Unreleased]`.

## Code of Conduct

This project follows the [Contributor Covenant](CODE_OF_CONDUCT.md).
Be respectful; assume good faith.

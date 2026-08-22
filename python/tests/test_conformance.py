"""Runs the shared cross-SDK conformance fixtures against the Python SDK.

The fixtures in spec/conformance/ are language-neutral: every SDK runs the
same cases so a behavior that drifts in one lane fails there. See
spec/conformance/README.md for the fixture schema and for the behaviors
that are deliberately language-specific (and therefore not pinned here).
"""

from __future__ import annotations

import json
import pathlib

import pytest

import simsys_logevent as m

SPEC_DIR = pathlib.Path(__file__).resolve().parents[2] / "spec" / "conformance"
CASES = sorted(SPEC_DIR.glob("*.json"))


def _load(path: pathlib.Path) -> dict:
    return json.loads(path.read_text(encoding="utf-8"))


def test_fixture_directory_is_not_empty():
    """A glob that silently matches nothing would make every case below
    vacuously pass. Fail loudly instead."""
    assert CASES, f"no conformance fixtures found under {SPEC_DIR}"


@pytest.mark.parametrize("path", CASES, ids=lambda p: p.stem)
def test_conformance(path: pathlib.Path):
    case = _load(path)
    expect = case["expect"]
    cfg = case.get("configure")
    lines: list[str] = []

    # Reset module state so cases cannot leak into each other.
    m._service = "unknown"
    m._default_level = "info"
    m._out = lines.append

    if cfg is not None:
        kwargs = {"service": cfg["service"], "out": lines.append}
        if "defaultLevel" in cfg:
            kwargs["default_level"] = cfg["defaultLevel"]

        if expect.get("configure_error"):
            with pytest.raises(ValueError):
                m.configure(**kwargs)
            return
        m.configure(**kwargs)
    else:
        assert not expect.get("configure_error"), (
            "a case with configure: null cannot expect a configure error"
        )

    call = case.get("call")
    if call is None:
        return

    fields = dict(call.get("fields", {}))
    if call["fn"] == "logEvent":
        if call.get("level") is not None:
            fields["level"] = call["level"]
        m.log_event(call["event"], **fields)
    elif call["fn"] == "logError":
        m.log_error(call["event"], ValueError(call["error_message"]), **fields)
    else:  # pragma: no cover - guards against a fixture typo
        pytest.fail(f"unknown fixture fn: {call['fn']}")

    if not expect["emitted"]:
        assert lines == [], f"expected no output, got {lines}"
        return

    assert len(lines) == 1, f"expected exactly one line, got {len(lines)}"
    got = json.loads(lines[0])

    for key, want in expect.get("fields", {}).items():
        assert got.get(key) == want, f"{key}: got {got.get(key)!r}, want {want!r}"
    for key in expect.get("present", []):
        assert key in got, f"expected field {key} to be present"
    for key, unwanted in expect.get("not_fields", {}).items():
        assert got.get(key) != unwanted, f"{key} must not be {unwanted!r}"

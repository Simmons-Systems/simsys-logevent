import json
import pytest
from simsys_logevent import configure, log_event, log_error, get_service


@pytest.fixture(autouse=True)
def _reset():
    import simsys_logevent as m
    m._service = "unknown"
    m._default_level = "info"
    yield


def _capture():
    lines = []
    configure(service="test-svc", out=lambda line: lines.append(line))
    return lines


def test_emits_one_json_line():
    lines = _capture()
    log_event("demo.event")
    assert len(lines) == 1
    parsed = json.loads(lines[0])
    assert parsed["event"] == "demo.event"


def test_includes_core_fields():
    lines = _capture()
    log_event("demo.event")
    parsed = json.loads(lines[0])
    assert parsed["ts"].endswith("Z")
    assert parsed["level"] == "info"
    assert parsed["level_code"] == 2
    assert parsed["service"] == "test-svc"
    assert isinstance(parsed["hostname"], str)
    assert isinstance(parsed["pid"], int)


def test_respects_level_override():
    lines = _capture()
    log_event("boom", level="error")
    parsed = json.loads(lines[0])
    assert parsed["level"] == "error"
    assert parsed["level_code"] == 4


def test_preserves_extra_fields():
    lines = _capture()
    log_event("shift.assigned", user="alice", outcome="success", shift_id="abc")
    parsed = json.loads(lines[0])
    assert parsed["user"] == "alice"
    assert parsed["outcome"] == "success"
    assert parsed["shift_id"] == "abc"


def test_no_emit_on_missing_event():
    lines = _capture()
    log_event("")
    log_event(None)
    assert len(lines) == 0


def test_never_throws():
    lines = _capture()
    log_event("demo", bad_value=object())
    assert len(lines) == 1


def test_level_codes():
    lines = _capture()
    for level, code in [("debug", 1), ("info", 2), ("warn", 3), ("error", 4)]:
        lines.clear()
        log_event("demo", level=level)
        assert json.loads(lines[0])["level_code"] == code


def test_log_error_extracts_fields():
    lines = _capture()
    try:
        raise TypeError("connection reset")
    except TypeError as e:
        log_error("db.query.failed", e)
    parsed = json.loads(lines[0])
    assert parsed["event"] == "db.query.failed"
    assert parsed["level"] == "error"
    assert parsed["error_type"] == "TypeError"
    assert parsed["error_message"] == "connection reset"
    assert "Traceback" in parsed["stack"]


def test_log_error_with_non_exception():
    lines = _capture()
    log_error("unexpected", "some string")
    parsed = json.loads(lines[0])
    assert parsed["error_message"] == "some string"
    assert "error_type" not in parsed


def test_log_error_with_extra_fields():
    lines = _capture()
    log_error("api.failed", ValueError("timeout"), route="/api/data")
    parsed = json.loads(lines[0])
    assert parsed["route"] == "/api/data"


def test_configure_requires_service():
    with pytest.raises(ValueError, match="service"):
        configure(service="")


def test_get_service():
    _capture()
    assert get_service() == "test-svc"

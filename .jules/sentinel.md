## 2026-07-01 - Prevent Log Spoofing / Metadata Overwrite
**Vulnerability:** User-provided fields in log events could overwrite critical system metadata (like `service`, `ts`, `pid`, `hostname`, `level_code`).
**Learning:** This occurred because the user-supplied payload was being merged into the log object *after* the system fields were set across all three languages (Node, Python, Go).
**Prevention:** Always ensure system-defined security metadata takes precedence when constructing objects or maps that include untrusted user input, by merging user inputs *before* setting system fields.

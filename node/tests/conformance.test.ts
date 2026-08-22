/**
 * Runs the shared cross-SDK conformance fixtures against the Node SDK.
 *
 * The fixtures in spec/conformance/ are language-neutral: every SDK runs
 * the same cases so a behavior that drifts in one lane fails there. See
 * spec/conformance/README.md for the fixture schema and for the behaviors
 * that are deliberately language-specific (and therefore not pinned).
 */
import { describe, expect, it, vi } from "vitest";
import { readdirSync, readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join, resolve } from "node:path";

const SPEC_DIR = resolve(
  dirname(fileURLToPath(import.meta.url)),
  "../../spec/conformance",
);

interface Fixture {
  name: string;
  description: string;
  configure: { service: string; defaultLevel?: string } | null;
  call?: {
    fn: "logEvent" | "logError";
    event: string;
    level?: string;
    error_message?: string;
    fields?: Record<string, unknown>;
  };
  expect: {
    emitted?: boolean;
    configure_error?: boolean;
    fields?: Record<string, unknown>;
    present?: string[];
    not_fields?: Record<string, unknown>;
  };
}

const files = readdirSync(SPEC_DIR)
  .filter((f: string) => f.endsWith(".json"))
  .sort();

const fixtures: Fixture[] = files.map(
  (f: string) => JSON.parse(readFileSync(join(SPEC_DIR, f), "utf8")) as Fixture,
);

describe("cross-SDK conformance", () => {
  it("finds fixtures", () => {
    // A glob matching nothing would make every case below vacuously pass.
    expect(fixtures.length).toBeGreaterThan(0);
  });

  for (const fx of fixtures) {
    it(fx.name, async () => {
      // Fresh module instance per case so module-level state cannot leak.
      vi.resetModules();
      const mod = await import("../src/index.js");

      const lines: string[] = [];
      let restoreStdout: (() => void) | undefined;

      if (fx.configure !== null) {
        const opts: Record<string, unknown> = {
          service: fx.configure.service,
          out: (line: string) => lines.push(line),
        };
        if (fx.configure.defaultLevel !== undefined) {
          opts.defaultLevel = fx.configure.defaultLevel;
        }

        if (fx.expect.configure_error) {
          expect(() => mod.configure(opts as never)).toThrow();
          return;
        }
        mod.configure(opts as never);
      } else {
        // `configure: null` means "observe the module in its default state",
        // so the injectable sink is unavailable by definition — setting it
        // would require the configure() call this case exists to skip.
        // Capturing the default stdout sink is the only faithful option.
        const spy = vi
          .spyOn(process.stdout, "write")
          .mockImplementation((chunk: unknown) => {
            lines.push(String(chunk).replace(/\n$/, ""));
            return true;
          });
        restoreStdout = () => spy.mockRestore();
      }

      try {
        const call = fx.call;
        if (!call) return;

        const fields = { ...(call.fields ?? {}) };
        if (call.fn === "logEvent") {
          const payload: Record<string, unknown> = {
            ...fields,
            event: call.event,
          };
          if (call.level !== undefined) payload.level = call.level;
          mod.logEvent(payload as never);
        } else {
          mod.logError(
            call.event,
            new Error(call.error_message ?? ""),
            fields as never,
          );
        }
      } finally {
        restoreStdout?.();
      }

      if (fx.expect.emitted === false) {
        expect(lines).toEqual([]);
        return;
      }

      expect(lines).toHaveLength(1);
      const got = JSON.parse(lines[0]) as Record<string, unknown>;

      for (const [key, want] of Object.entries(fx.expect.fields ?? {})) {
        expect(got[key], `field ${key}`).toEqual(want);
      }
      for (const key of fx.expect.present ?? []) {
        expect(got, `field ${key} present`).toHaveProperty(key);
      }
      for (const [key, unwanted] of Object.entries(fx.expect.not_fields ?? {})) {
        expect(got[key], `field ${key} must not be spoofed`).not.toEqual(
          unwanted,
        );
      }
    });
  }
});

"use client";

import { useState } from "react";
import { ChevronDown, ChevronRight, Download, Eye, EyeOff, Trash2, X } from "lucide-react";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Select } from "@/components/ui/Select";
import { Input } from "@/components/ui/input";
import { DashedAddButton, EmptyState, FieldLabel, Hint, inputCls } from "./Common";
import { useResourceProviders } from "./useResourceProviders";
import type { ContainerHelpers } from "./types";

export function ResourcesModalContent({
  container,
  cIndex,
  helpers,
}: {
  container: any;
  cIndex: number;
  helpers: ContainerHelpers;
}) {
  const { providers, loading } = useResourceProviders();
  const resources = container.resources || [];
  const [drafts, setDrafts] = useState<Record<number, string>>({});
  const [expanded, setExpanded] = useState<Record<number, boolean>>({});
  const [showPasswords, setShowPasswords] = useState<Record<string, boolean>>({});

  const updateResource = (i: number, key: string, val: any) =>
    helpers.updateArrayItem(cIndex, "resources", i, key, val);

  const toggleExpand = (i: number) => {
    setExpanded((prev) => ({
      ...prev,
      [i]: !prev[i],
    }));
  };

  const togglePasswordVisibility = (fieldKey: string) => {
    setShowPasswords((prev) => ({
      ...prev,
      [fieldKey]: !prev[fieldKey],
    }));
  };

  const addFilesItem = (
    i: number,
    key: string,
    items: string[],
    draft: string,
  ) => {
    const value = draft.trim();
    if (!value) return;
    updateResource(i, key, [...items, value]);
    setDrafts({ ...drafts, [i]: "" });
  };

  return (
    <div className="space-y-4">
      {loading && providers.length === 0 && (
        <Hint>Loading available resource providers…</Hint>
      )}
      {resources.length === 0 ? (
        <EmptyState
          icon={<Download className="size-8 text-[var(--dm-card-border)]" />}
          text="No external resources configured."
        />
      ) : (
        <div className="space-y-2.5">
          {resources.map((res: any, i: number) => {
            const provider = providers.find((p) => p.id === res.type);
            const isExpanded = expanded[i] ?? false;
            const summary =
              res.target ||
              res.url ||
              res.bucket ||
              (res.content
                ? res.content.length > 30
                  ? res.content.slice(0, 30).trim() + "…"
                  : res.content.trim()
                : "") ||
              "";

            return (
              <div
                key={i}
                className="overflow-hidden rounded-xl border border-[var(--dm-card-border)] bg-[var(--dm-card-2)] transition"
              >
                {/* Collapsible Card Header */}
                <div
                  className="flex items-center justify-between gap-3 p-3.5 cursor-pointer select-none transition hover:bg-[var(--dm-card-hover)]"
                  onClick={() => toggleExpand(i)}
                >
                  <div className="flex items-center gap-2.5 min-w-0 flex-1">
                    <div className="text-[var(--text-muted)] hover:text-[var(--text-main)] transition-colors p-0.5 shrink-0">
                      {isExpanded ? (
                        <ChevronDown className="size-4 shrink-0" />
                      ) : (
                        <ChevronRight className="size-4 shrink-0" />
                      )}
                    </div>
                    <Badge
                      variant="default"
                      className="shrink-0 font-medium px-2 py-0.5 text-[11px] border-[var(--dm-card-border)] bg-[var(--dm-inset)]"
                    >
                      {provider?.name || res.type || "Resource"}
                    </Badge>
                    <span className="truncate text-xs font-mono text-[var(--text-muted)]">
                      {summary ? (
                        summary
                      ) : (
                        <span className="italic text-[var(--text-muted)]/70">
                          (Click to configure resource)
                        </span>
                      )}
                    </span>
                  </div>
                  <div
                    className="flex items-center gap-1.5 shrink-0"
                    onClick={(e) => e.stopPropagation()}
                  >
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => {
                        helpers.removeArrayItem(cIndex, "resources", i);
                        setExpanded((prev) => {
                          const next = { ...prev };
                          delete next[i];
                          return next;
                        });
                      }}
                      className="border-[var(--dm-card-border)] text-[var(--text-muted)] hover:border-red-500/50 hover:text-red-400"
                    >
                      <Trash2 className="size-3" />
                    </Button>
                  </div>
                </div>

                {/* Collapsible Card Body */}
                {isExpanded && (
                  <div className="space-y-4 border-t border-[var(--dm-divider)] p-4 pt-3.5">
                    {/* Explicit Provider Selector with FieldLabel */}
                    <div className="space-y-1.5">
                      <FieldLabel>Resource Provider</FieldLabel>
                      <div className="w-full sm:w-64">
                        <Select
                          value={res.type}
                          onChange={(val: string) => updateResource(i, "type", val)}
                          options={providers.map((p) => ({
                            value: p.id,
                            label: p.name,
                          }))}
                          placeholder="Select provider..."
                        />
                      </div>
                    </div>

                    {provider ? (
                      <div className="space-y-4">
                        {provider.description && (
                          <Hint>{provider.description}</Hint>
                        )}
                        {provider.fields.map((field) => {
                          const key = field.spec_key || field.name;
                          const value = res[key];
                          const pwdKey = `${i}-${field.name}`;
                          const isPassword = field.type === "password" || field.type === "secret";

                          if (field.type === "string_list") {
                            const storedItems: string[] = Array.isArray(value)
                              ? value.filter(Boolean)
                              : value
                                ? String(value)
                                    .split(",")
                                    .map((f: string) => f.trim())
                                    .filter(Boolean)
                                : [];
                            const draft = drafts[i] ?? "";
                            const addItem = () => {
                              const trimmed = draft.trim();
                              if (!trimmed) return;
                              addFilesItem(i, key, storedItems, trimmed);
                            };
                            const removeItem = (idx: number) => {
                              const newItems = storedItems.filter(
                                (_, j) => j !== idx,
                              );
                              updateResource(i, key, newItems);
                            };
                            return (
                              <div key={field.name} className="space-y-1.5">
                                <FieldLabel>
                                  {field.label}
                                  {!field.required && (
                                    <span className="text-[var(--text-muted)]">
                                      {" "}
                                      (Optional)
                                    </span>
                                  )}
                                </FieldLabel>
                                <div className="flex flex-wrap gap-2 mb-2">
                                  {storedItems.map((item, idx) => (
                                    <Badge
                                      key={idx}
                                      variant="default"
                                      className="flex items-center gap-1 px-2 py-0.5 text-xs font-mono"
                                    >
                                      {item}
                                      <X
                                        className="size-3 cursor-pointer"
                                        onClick={() => removeItem(idx)}
                                      />
                                    </Badge>
                                  ))}
                                </div>
                                <Input
                                  value={draft}
                                  onChange={(e) =>
                                    setDrafts({ ...drafts, [i]: e.target.value })
                                  }
                                  onKeyDown={(e) => {
                                    if (e.key === "Enter") {
                                      e.preventDefault();
                                      addItem();
                                    }
                                  }}
                                  placeholder={field.placeholder}
                                  className={`${inputCls} font-mono`}
                                />
                                {field.description && (
                                  <Hint>{field.description}</Hint>
                                )}
                              </div>
                            );
                          }

                          return (
                            <div key={field.name} className="space-y-1.5">
                              <FieldLabel>
                                {field.label}
                                {!field.required && (
                                  <span className="text-[var(--text-muted)]">
                                    {" "}
                                    (Optional)
                                  </span>
                                )}
                              </FieldLabel>
                              {field.type === "textarea" ? (
                                <textarea
                                  value={value ?? ""}
                                  onChange={(e) =>
                                    updateResource(i, key, e.target.value)
                                  }
                                  placeholder={field.placeholder}
                                  rows={4}
                                  className="w-full rounded-lg border border-[var(--dm-card-border)] bg-[var(--dm-input)] px-3 py-2 text-xs font-mono text-[var(--text-main)] outline-none transition-colors resize-y focus-visible:border-blue-500 focus-visible:ring-1 focus-visible:ring-blue-500/30"
                                />
                              ) : isPassword ? (
                                <div className="relative">
                                  <Input
                                    type={showPasswords[pwdKey] ? "text" : "password"}
                                    value={value ?? ""}
                                    onChange={(e) =>
                                      updateResource(i, key, e.target.value)
                                    }
                                    placeholder={field.placeholder}
                                    className={`${inputCls} pr-10 font-mono`}
                                  />
                                  <button
                                    type="button"
                                    onClick={() => togglePasswordVisibility(pwdKey)}
                                    className="absolute right-2.5 top-1/2 -translate-y-1/2 text-[var(--text-muted)] hover:text-[var(--text-main)] transition-colors p-1"
                                    title={showPasswords[pwdKey] ? "Hide secret" : "Show secret"}
                                  >
                                    {showPasswords[pwdKey] ? (
                                      <EyeOff className="size-3.5" />
                                    ) : (
                                      <Eye className="size-3.5" />
                                    )}
                                  </button>
                                </div>
                              ) : (
                                <Input
                                  type={field.type === "number" ? "number" : "text"}
                                  value={value ?? ""}
                                  onChange={(e) =>
                                    updateResource(i, key, e.target.value)
                                  }
                                  placeholder={field.placeholder}
                                  className={inputCls}
                                />
                              )}
                              {field.description && (
                                <Hint>{field.description}</Hint>
                              )}
                            </div>
                          );
                        })}
                      </div>
                    ) : (
                      <Hint>
                        {loading
                          ? "Loading provider fields…"
                          : `Unknown provider type "${res.type}".`}
                      </Hint>
                    )}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}
      <DashedAddButton
        disabled={loading}
        onClick={() => {
          const first = providers[0]?.id || "http";
          const newIdx = resources.length;
          helpers.addArrayItem(cIndex, "resources", { type: first });
          setExpanded((prev) => ({ ...prev, [newIdx]: true }));
        }}
        text="Add Resource"
      />
    </div>
  );
}

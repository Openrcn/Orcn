"use client";

import { useEffect, useState, useMemo, useRef, useCallback } from "react";
import { Plus, Trash2, GripVertical } from "lucide-react";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/input";

interface ArrayItemSchema {
  type: "object" | "string" | "number" | "boolean";
  properties?: Record<string, { type: string; label?: string; placeholder?: string }>;
  required?: string[];
}

interface JsonArrayInputProps {
  value: string;
  onChange: (value: string) => void;
  schema: ArrayItemSchema;
  addButtonLabel?: string;
  emptyMessage?: string;
  description?: string;
}

export function JsonArrayInput({
  value,
  onChange,
  schema,
  addButtonLabel = "Add Item",
  emptyMessage = "No items yet",
  description,
}: JsonArrayInputProps) {
  const [items, setItems] = useState<Array<Record<string, any>>>([]);
  const [initialized, setInitialized] = useState(false);
  const pendingChange = useRef<string | null>(null);
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;

  const itemFields = useMemo(() => {
    if (schema.type === "object" && schema.properties) {
      return Object.entries(schema.properties).map(([key, prop]) => ({
        key,
        label: prop.label || key,
        type: prop.type === "password" ? "password" : "text",
        placeholder: prop.placeholder || "",
        required: schema.required?.includes(key),
        autoGenerate: key === "id",
      }));
    }
    return [{ key: "value", label: "Value", type: schema.type, placeholder: "", required: false, autoGenerate: false }];
  }, [schema]);

  useEffect(() => {
    if (initialized) return;
    try {
      const parsed = value ? JSON.parse(value) : [];
      if (Array.isArray(parsed)) {
        if (schema.type === "object") {
          const withIds = parsed.map((item: any, idx: number) => {
            const obj: Record<string, any> = {};
            itemFields.forEach((f) => {
              if (f.autoGenerate && !item[f.key]) {
                obj[f.key] = `item_${Date.now()}_${idx}_${Math.random().toString(36).slice(2, 8)}`;
              } else {
                obj[f.key] = item[f.key] ?? "";
              }
            });
            return obj;
          });
          setItems(withIds);
        } else {
          setItems(parsed.map((v: any) => ({ value: v })));
        }
      }
    } catch {
      setItems([]);
    }
    setInitialized(true);
  }, [value, schema.type, itemFields]);

  // Flush pending change after render
  useEffect(() => {
    if (pendingChange.current !== null) {
      onChangeRef.current(pendingChange.current);
      pendingChange.current = null;
    }
  }, [items]);

  const scheduleChange = useCallback((newItems: Array<Record<string, any>>) => {
    let output: any[];
    if (schema.type === "object") {
      output = newItems;
    } else {
      output = newItems.map((item) => item.value);
    }
    pendingChange.current = JSON.stringify(output);
    setItems(newItems);
  }, [schema.type]);

  const addItem = useCallback(() => {
    setItems((prev) => {
      const newItem: Record<string, any> = {};
      if (schema.type === "object") {
        itemFields.forEach((f) => {
          if (f.autoGenerate) {
            newItem[f.key] = `item_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`;
          } else {
            newItem[f.key] = "";
          }
        });
      } else {
        newItem.value = "";
      }
      const next = [...prev, newItem];
      // Schedule change after state update
      setTimeout(() => {
        let output: any[];
        if (schema.type === "object") {
          output = next;
        } else {
          output = next.map((item) => item.value);
        }
        onChangeRef.current(JSON.stringify(output));
      }, 0);
      return next;
    });
  }, [schema.type, itemFields]);

  const updateItem = useCallback((index: number, key: string, val: any) => {
    setItems((prev) => {
      const next = [...prev];
      next[index] = { ...next[index], [key]: val };
      // Schedule change after state update
      setTimeout(() => {
        let output: any[];
        if (schema.type === "object") {
          output = next;
        } else {
          output = next.map((item) => item.value);
        }
        onChangeRef.current(JSON.stringify(output));
      }, 0);
      return next;
    });
  }, [schema.type]);

  const removeItem = useCallback((index: number) => {
    setItems((prev) => {
      const next = prev.filter((_, i) => i !== index);
      setTimeout(() => {
        let output: any[];
        if (schema.type === "object") {
          output = next;
        } else {
          output = next.map((item) => item.value);
        }
        onChangeRef.current(JSON.stringify(output));
      }, 0);
      return next;
    });
  }, [schema.type]);

  const renderItemInput = (item: Record<string, any>, index: number) => {
    if (schema.type === "object") {
      return (
        <div className="flex-1 grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 w-full min-w-0">
          {itemFields.map((field) => {
            if (field.autoGenerate) return null;
            return (
              <div key={field.key} className="space-y-1 w-full min-w-0">
                <label className="text-[10px] font-medium text-[var(--dm-text-3)]">
                  {field.label} {field.required && <span className="text-red-400">*</span>}
                </label>
                <Input
                  type={field.type || "text"}
                  placeholder={field.placeholder || field.key}
                  value={item[field.key] ?? ""}
                  onChange={(e) => updateItem(index, field.key, e.target.value)}
                  className="h-8 text-xs w-full min-w-0"
                />
              </div>
            );
          })}
        </div>
      );
    } else {
      return (
        <div className="flex-1 space-y-1 w-full min-w-0">
          <label className="text-[10px] font-medium text-[var(--dm-text-3)]">Value</label>
          <Input
            type={schema.type === "number" ? "number" : schema.type === "boolean" ? "checkbox" : "text"}
            placeholder="Enter value"
            value={item.value ?? ""}
            onChange={(e) => updateItem(index, "value", schema.type === "number" ? Number(e.target.value) : schema.type === "boolean" ? e.target.checked : e.target.value)}
            className="h-8 text-xs w-full min-w-0"
          />
        </div>
      );
    }
  };

  return (
    <div className="space-y-3 w-full min-w-0">
      {description && <p className="text-[10px] text-[var(--text-light)]">{description}</p>}
      {items.length === 0 ? (
        <div className="w-full text-center py-8 border-2 border-dashed border-[var(--dm-card-border)] rounded-lg min-w-0">
          <p className="text-xs text-[var(--text-muted)]">{emptyMessage}</p>
          <Button variant="outline" size="sm" className="mt-3" onClick={addItem}>
            <Plus className="mr-1.5 size-3.5" /> {addButtonLabel}
          </Button>
        </div>
      ) : (
        <div className="space-y-3 w-full min-w-0">
          {items.map((item, index) => (
            <div
              key={index}
              className="flex flex-col sm:flex-row gap-3 p-3 rounded-lg border border-[var(--dm-card-border)] bg-[var(--dm-inset)] w-full min-w-0"
            >
              <div className="flex items-center gap-2 text-[var(--text-muted)] shrink-0 pt-1">
                <GripVertical className="size-4 cursor-grab" />
                <span className="text-xs font-mono">{index + 1}</span>
              </div>
              {renderItemInput(item, index)}
              <Button
                variant="ghost"
                size="sm"
                className="self-start mt-6 sm:mt-0 text-red-400 hover:bg-red-500/10 shrink-0"
                onClick={() => removeItem(index)}
              >
                <Trash2 className="size-3.5" />
              </Button>
            </div>
          ))}
          <div className="w-full min-w-0">
            <Button variant="outline" size="sm" className="w-full" onClick={addItem}>
              <Plus className="mr-1.5 size-3.5" /> {addButtonLabel}
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
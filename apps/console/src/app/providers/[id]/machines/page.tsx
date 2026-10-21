"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { Loader2, Plus, Pencil, Trash2, CheckCircle, AlertCircle, RefreshCw, Server } from "lucide-react";
import { Button } from "@/components/ui/Button";
import {
  fetchConnectionMachines,
  createMachine,
  updateMachine,
  deleteMachine,
  type Machine,
} from "@/components/providers/api";

export default function MachinesPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const connectionId = String(params.id);

  const [machines, setMachines] = useState<Machine[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [busyId, setBusyId] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [editingMachine, setEditingMachine] = useState<Machine | null>(null);

  const [formName, setFormName] = useState("");
  const [formEndpoint, setFormEndpoint] = useState("");
  const [formProxyEndpoint, setFormProxyEndpoint] = useState("");
  const [formToken, setFormToken] = useState("");

  useEffect(() => {
    loadMachines();
  }, [connectionId]);

  async function loadMachines() {
    setLoading(true);
    try {
      const data = await fetchConnectionMachines(connectionId);
      setMachines(data.machines || []);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Failed to load machines");
    } finally {
      setLoading(false);
    }
  }

  function resetForm() {
    setFormName("");
    setFormEndpoint("");
    setFormProxyEndpoint("");
    setFormToken("");
    setEditingMachine(null);
    setShowForm(false);
  }

  function startEdit(machine: Machine) {
    setEditingMachine(machine);
    setFormName(machine.Name);
    setFormEndpoint(machine.Endpoint);
    setFormProxyEndpoint(machine.ProxyEndpoint || "");
    setFormToken("");
    setShowForm(true);
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    const id = editingMachine?.ID;

    if (!formName.trim() || !formEndpoint.trim() || (!id && !formToken.trim())) {
      setError("Name, endpoint, and token are required");
      return;
    }

    setBusyId(id || "new");
    try {
      if (id) {
        await updateMachine(connectionId, id, {
          name: formName,
          endpoint: formEndpoint,
          proxy_endpoint: formProxyEndpoint,
          token: formToken || undefined,
        });
      } else {
        await createMachine(connectionId, {
          name: formName,
          endpoint: formEndpoint,
          proxy_endpoint: formProxyEndpoint,
          token: formToken,
        });
      }
      resetForm();
      await loadMachines();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Failed to save machine");
    } finally {
      setBusyId(null);
    }
  }

  async function handleDelete(id: string) {
    if (!confirm("Delete this machine?")) return;
    setBusyId(id);
    setError("");
    try {
      await deleteMachine(connectionId, id);
      await loadMachines();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Failed to delete machine");
    } finally {
      setBusyId(null);
    }
  }

  function getStatusIcon(status: string) {
    switch (status) {
      case "healthy":
        return <CheckCircle className="size-3.5 text-emerald-400" />;
      case "unauthorized":
        return <AlertCircle className="size-3.5 text-amber-400" />;
      case "unreachable":
      case "unhealthy":
        return <AlertCircle className="size-3.5 text-red-400" />;
      default:
        return <RefreshCw className="size-3.5 text-[var(--text-muted)] animate-spin" />;
    }
  }

  function statusLabel(status: string) {
    switch (status) {
      case "healthy":
        return "Active";
      case "unauthorized":
        return "Unauthorized";
      case "unreachable":
        return "Unreachable";
      case "unhealthy":
        return "Unhealthy";
      default:
        return "Unknown";
    }
  }

  return (
    <div className="mx-auto max-w-4xl space-y-6">
      <div className="flex items-center gap-1.5 text-xs text-[var(--text-light)]">
        <button
          type="button"
          onClick={() => router.push("/providers")}
          className="transition-colors hover:text-[var(--dm-text-3)]"
        >
          Providers
        </button>
        <span>/</span>
        <span className="text-[var(--dm-text-3)]">Machines</span>
      </div>

      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-[var(--text-main)]">
            Machines
          </h1>
          <p className="mt-1 text-xs text-[var(--text-muted)]">
            Manage node-agent machines for this local provider connection.
          </p>
        </div>
        <Button size="sm" onClick={() => { resetForm(); setShowForm(true); }}>
          <Plus className="mr-1.5 size-3.5" /> Add Machine
        </Button>
      </div>

      {error && (
        <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-xs text-red-400">
          {error}
        </div>
      )}

      {showForm && (
        <div className="rounded-xl border border-[var(--dm-card-border)] bg-[var(--card-bg)] p-4">
          <h3 className="text-sm font-medium text-[var(--text-main)]">
            {editingMachine ? "Edit Machine" : "Add Machine"}
          </h3>
          <form onSubmit={handleSubmit} className="mt-4 space-y-4">
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div className="space-y-1.5">
                <label className="block text-xs font-medium text-[var(--dm-text-3)]">
                  Name <span className="text-red-400">*</span>
                </label>
                <input
                  value={formName}
                  onChange={(e) => setFormName(e.target.value)}
                  placeholder="gpu-server-1"
                  className="h-9 w-full rounded-lg border border-[var(--border)] bg-[var(--dm-inset)] px-3 text-xs text-[var(--text-main)] placeholder:text-[var(--text-light)] focus:outline-none focus:border-blue-500"
                />
              </div>
              <div className="space-y-1.5">
                <label className="block text-xs font-medium text-[var(--dm-text-3)]">
                  Endpoint <span className="text-red-400">*</span>
                </label>
                <input
                  value={formEndpoint}
                  onChange={(e) => setFormEndpoint(e.target.value)}
                  placeholder="http://192.168.1.10:9090"
                  className="h-9 w-full rounded-lg border border-[var(--border)] bg-[var(--dm-inset)] px-3 text-xs text-[var(--text-main)] placeholder:text-[var(--text-light)] focus:outline-none focus:border-blue-500"
                />
              </div>
              <div className="space-y-1.5 sm:col-span-2">
                <label className="block text-xs font-medium text-[var(--dm-text-3)]">
                  Proxy Endpoint
                </label>
                <input
                  value={formProxyEndpoint}
                  onChange={(e) => setFormProxyEndpoint(e.target.value)}
                  placeholder="http://192.168.1.10:8080"
                  className="h-9 w-full rounded-lg border border-[var(--border)] bg-[var(--dm-inset)] px-3 text-xs text-[var(--text-main)] placeholder:text-[var(--text-light)] focus:outline-none focus:border-blue-500"
                />
                <p className="text-[10px] text-[var(--text-light)]">
                  Node-agent proxy used for public workload URLs. Required to expose
                  workloads on the internet.
                </p>
              </div>
            </div>
            <div className="space-y-1.5">
              <label className="block text-xs font-medium text-[var(--dm-text-3)]">
                Token <span className="text-red-400">*</span>
              </label>
              <input
                type={editingMachine ? "password" : "password"}
                value={formToken}
                onChange={(e) => setFormToken(e.target.value)}
                placeholder={editingMachine ? "Leave empty to keep current" : "Admin token from node-agent"}
                className="h-9 w-full rounded-lg border border-[var(--border)] bg-[var(--dm-inset)] px-3 text-xs text-[var(--text-main)] placeholder:text-[var(--text-light)] focus:outline-none focus:border-blue-500"
              />
              {editingMachine && (
                <p className="text-[10px] text-[var(--text-light)]">
                  Leave empty to keep the current token.
                </p>
              )}
            </div>
            <div className="flex justify-end gap-2 pt-2 border-t border-[var(--dm-divider)]">
              <Button type="button" variant="secondary" size="sm" onClick={resetForm}>
                Cancel
              </Button>
              <Button type="submit" size="sm" disabled={busyId !== null}>
                {busyId ? (
                  <>
                    <Loader2 className="mr-1.5 size-3.5 animate-spin" />
                    Saving…
                  </>
                ) : (
                  "Save"
                )}
              </Button>
            </div>
          </form>
        </div>
      )}

      {loading ? (
        <div className="flex items-center justify-center py-16 text-xs text-[var(--text-muted)]">
          <Loader2 className="mr-2 size-4 animate-spin" />
          Loading machines…
        </div>
      ) : machines.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-xl border border-dashed border-[var(--dm-card-border)] bg-[var(--card-bg)] py-16 text-center">
          <div className="flex h-11 w-11 items-center justify-center rounded-xl border border-[var(--border)] bg-[var(--dm-inset)]">
            <Server className="size-5 text-[var(--text-muted)]" />
          </div>
          <p className="mt-3 text-sm font-medium text-[var(--text-main)]">
            No machines added yet
          </p>
          <p className="mt-1 text-xs text-[var(--text-muted)]">
            Add a node-agent machine to start deploying workloads.
          </p>
          <Button size="sm" className="mt-4" onClick={() => { resetForm(); setShowForm(true); }}>
            <Plus className="mr-1.5 size-3.5" /> Add Machine
          </Button>
        </div>
      ) : (
        <div className="rounded-xl border border-[var(--dm-card-border)] bg-[var(--card-bg)] overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead>
                <tr className="border-b border-[var(--dm-divider)] bg-[var(--dm-inset)] text-xs text-[var(--text-muted)]">
                  <th className="p-3 font-medium">Name</th>
                  <th className="p-3 font-medium">Endpoint</th>
                  <th className="p-3 font-medium">Status</th>
                  <th className="p-3 font-medium">Last Seen</th>
                  <th className="p-3 font-medium text-right">Actions</th>
                </tr>
              </thead>
              <tbody>
                {machines.map((m) => (
                  <tr key={m.ID} className="border-b border-[var(--dm-divider)] last:border-0">
                    <td className="p-3 font-medium text-[var(--text-main)]">{m.Name}</td>
                    <td className="p-3 text-[var(--dm-text-3)] font-mono text-xs">{m.Endpoint}</td>
                    <td className="p-3">
                      <div className="flex items-center gap-1.5">
                        {getStatusIcon(m.Status)}
                        <span className="text-xs">{statusLabel(m.Status)}</span>
                      </div>
                    </td>
                    <td className="p-3 text-[var(--dm-text-3)] text-xs">
                      {m.LastSeen ? new Date(m.LastSeen).toLocaleString() : "Never"}
                    </td>
                    <td className="p-3 text-right">
                      <div className="flex items-center justify-end gap-1">
                        <button
                          type="button"
                          onClick={() => startEdit(m)}
                          disabled={busyId === m.ID}
                          title="Edit"
                          className="flex items-center justify-center rounded-md border border-[var(--border)] p-1.5 text-[var(--text-muted)] transition-colors hover:border-[var(--border-hover)] hover:text-[var(--text-main)] disabled:opacity-50"
                        >
                          <Pencil className="size-3.5" />
                        </button>
                        <button
                          type="button"
                          onClick={() => handleDelete(m.ID)}
                          disabled={busyId === m.ID}
                          title="Delete"
                          className="flex items-center justify-center rounded-md border border-[var(--border)] p-1.5 text-[var(--text-muted)] transition-colors hover:border-red-500/40 hover:text-red-400 disabled:opacity-50"
                        >
                          <Trash2 className="size-3.5" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
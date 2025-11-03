// src/lib/api.js
const BASE = "/api";

async function req(path, { method = "GET", body, headers } = {}) {
  const res = await fetch(`${BASE}${path}`, {
    method,
    headers: { "Content-Type": "application/json", ...(headers || {}) },
    body: body ? JSON.stringify(body) : undefined,
    credentials: "include",
  });
  if (!res.ok) throw new Error(`${res.status} ${await res.text().catch(() => "")}`);
  const ct = res.headers.get("content-type") || "";
  return ct.includes("application/json") ? res.json() : {};
}

export function unlock(vault, master) {
  return req("/unlock", { method: "POST", body: { vault, master } });
}
export function lock() {
  return req("/lock", { method: "POST" });
}
export async function listItems(params = {}) {
  const qs = new URLSearchParams(params);
  const data = await req(`/items${qs.toString() ? `?${qs}` : ""}`);
  return Array.isArray(data) ? { items: data } : (data || { items: [] });
}
export function getItem(id) {
  return req(`/items/${id}`);
}
export function createItem(item) {
  return req("/items", { method: "POST", body: item });
}
export function updateItem(id, patch) {
  return req(`/items/${id}`, { method: "PUT", body: patch });
}
export function deleteItem(id) {
  return req(`/items/${id}`, { method: "DELETE" });
}

// (optional) also export a default object if you like
const api = { unlock, lock, listItems, getItem, createItem, updateItem, deleteItem };
export default api;

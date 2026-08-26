CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS tenants (id TEXT PRIMARY KEY, name TEXT NOT NULL, status TEXT NOT NULL, created_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS users (id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL REFERENCES tenants(id), email TEXT NOT NULL UNIQUE, password_hash TEXT NOT NULL, role TEXT NOT NULL, status TEXT NOT NULL, created_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS sessions (id TEXT PRIMARY KEY, user_id TEXT NOT NULL REFERENCES users(id), token_hash TEXT NOT NULL UNIQUE, expires_at TEXT NOT NULL, revoked_at TEXT, created_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS vessels (id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL REFERENCES tenants(id), imo TEXT NOT NULL, name TEXT NOT NULL, flag TEXT NOT NULL, deadweight_kg REAL NOT NULL, status TEXT NOT NULL, created_at TEXT NOT NULL, UNIQUE(tenant_id, imo));
CREATE TABLE IF NOT EXISTS vessel_certificates (id TEXT PRIMARY KEY, vessel_id TEXT NOT NULL REFERENCES vessels(id), number TEXT NOT NULL, expires_at TEXT NOT NULL, verified INTEGER NOT NULL, created_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS terminals (id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL REFERENCES tenants(id), name TEXT NOT NULL, timezone TEXT NOT NULL, open_from TEXT NOT NULL, open_until TEXT NOT NULL, status TEXT NOT NULL, created_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS fuel_lots (id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL REFERENCES tenants(id), lot_number TEXT NOT NULL, product TEXT NOT NULL, available_kg REAL NOT NULL, quality_state TEXT NOT NULL, received_at TEXT NOT NULL, UNIQUE(tenant_id, lot_number));
CREATE TABLE IF NOT EXISTS bunker_windows (id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL REFERENCES tenants(id), terminal_id TEXT NOT NULL REFERENCES terminals(id), starts_at TEXT NOT NULL, ends_at TEXT NOT NULL, status TEXT NOT NULL, owner_id TEXT, version INTEGER NOT NULL, created_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS transfer_orders (id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL REFERENCES tenants(id), vessel_id TEXT NOT NULL REFERENCES vessels(id), window_id TEXT NOT NULL REFERENCES bunker_windows(id), fuel_lot_id TEXT NOT NULL REFERENCES fuel_lots(id), target_kg REAL NOT NULL, transferred_kg REAL NOT NULL, state TEXT NOT NULL, lease_owner TEXT, lease_until TEXT, version INTEGER NOT NULL, idempotency_key TEXT, created_at TEXT NOT NULL, UNIQUE(tenant_id, idempotency_key));
CREATE TABLE IF NOT EXISTS transfer_steps (id TEXT PRIMARY KEY, order_id TEXT NOT NULL REFERENCES transfer_orders(id), position INTEGER NOT NULL, name TEXT NOT NULL, status TEXT NOT NULL, confirmed_at TEXT, UNIQUE(order_id, position));
CREATE TABLE IF NOT EXISTS safety_permits (id TEXT PRIMARY KEY, order_id TEXT NOT NULL UNIQUE REFERENCES transfer_orders(id), status TEXT NOT NULL, issued_by TEXT, created_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS samples (id TEXT PRIMARY KEY, order_id TEXT NOT NULL REFERENCES transfer_orders(id), chain_ref TEXT NOT NULL UNIQUE, receiver TEXT NOT NULL, quality_state TEXT NOT NULL, created_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS custody_events (id TEXT PRIMARY KEY, sample_id TEXT NOT NULL REFERENCES samples(id), actor_id TEXT NOT NULL REFERENCES users(id), action TEXT NOT NULL, created_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS invoices (id TEXT PRIMARY KEY, order_id TEXT NOT NULL UNIQUE REFERENCES transfer_orders(id), amount_cents INTEGER NOT NULL, currency TEXT NOT NULL, state TEXT NOT NULL, payment_key TEXT, created_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS outbox_messages (id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL REFERENCES tenants(id), topic TEXT NOT NULL, payload TEXT NOT NULL, status TEXT NOT NULL, attempts INTEGER NOT NULL, next_attempt TEXT NOT NULL, lease_owner TEXT, lease_until TEXT, last_error TEXT, created_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS audit_events (id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL REFERENCES tenants(id), actor_id TEXT, action TEXT NOT NULL, object_id TEXT NOT NULL, request_id TEXT NOT NULL, created_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS idempotency_keys (id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL REFERENCES tenants(id), key_value TEXT NOT NULL, request_hash TEXT NOT NULL, response_code INTEGER NOT NULL, response_body TEXT NOT NULL, created_at TEXT NOT NULL, UNIQUE(tenant_id, key_value));
CREATE TABLE IF NOT EXISTS incidents (id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL REFERENCES tenants(id), order_id TEXT NOT NULL REFERENCES transfer_orders(id), severity TEXT NOT NULL, status TEXT NOT NULL, summary TEXT NOT NULL, created_at TEXT NOT NULL);
CREATE INDEX IF NOT EXISTS idx_vessels_tenant ON vessels(tenant_id, created_at);
CREATE INDEX IF NOT EXISTS idx_windows_tenant_time ON bunker_windows(tenant_id, starts_at, ends_at);
CREATE INDEX IF NOT EXISTS idx_lots_tenant_quality ON fuel_lots(tenant_id, quality_state);
CREATE INDEX IF NOT EXISTS idx_orders_tenant_state ON transfer_orders(tenant_id, state, created_at);
CREATE INDEX IF NOT EXISTS idx_outbox_due ON outbox_messages(status, next_attempt, lease_until);
CREATE INDEX IF NOT EXISTS idx_audit_tenant_time ON audit_events(tenant_id, created_at, id);
INSERT OR IGNORE INTO schema_migrations(version, applied_at) VALUES (1, CURRENT_TIMESTAMP);
INSERT OR IGNORE INTO tenants(id, name, status, created_at) VALUES ('tenant-zj', 'Zhejiang Green Marine', 'active', CURRENT_TIMESTAMP), ('tenant-fj', 'Fujian Bunkering', 'active', CURRENT_TIMESTAMP);
INSERT OR IGNORE INTO users(id, tenant_id, email, password_hash, role, status, created_at) VALUES
 ('user-planner', 'tenant-zj', 'planner@example.test', 'planner-pass', 'planner', 'active', CURRENT_TIMESTAMP),
 ('user-quality', 'tenant-zj', 'quality@example.test', 'quality-pass', 'quality', 'active', CURRENT_TIMESTAMP),
 ('user-finance', 'tenant-zj', 'finance@example.test', 'finance-pass', 'finance', 'active', CURRENT_TIMESTAMP),
 ('user-fj', 'tenant-fj', 'planner-fj@example.test', 'planner-fj-pass', 'planner', 'active', CURRENT_TIMESTAMP);

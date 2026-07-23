-- ============================================================
-- AI Camera Monitoring System - Database Schema (PostgreSQL)
-- ============================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ---------------------------------------------------------------
-- USERS
-- ---------------------------------------------------------------
CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username      VARCHAR(64) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    ruolo         VARCHAR(32) NOT NULL DEFAULT 'operator', -- admin | operator
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ---------------------------------------------------------------
-- CAMERAS
-- ---------------------------------------------------------------
CREATE TABLE IF NOT EXISTS cameras (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    codice      VARCHAR(64) UNIQUE NOT NULL,   -- es. "camera_01"
    nome        VARCHAR(128) NOT NULL,          -- es. "Ingresso principale"
    posizione   VARCHAR(128),
    stato       VARCHAR(16) NOT NULL DEFAULT 'offline', -- online | offline
    last_seen   TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ---------------------------------------------------------------
-- ZONES (aree virtuali per lo zone monitoring)
-- ---------------------------------------------------------------
CREATE TABLE IF NOT EXISTS zones (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    camera_id   UUID NOT NULL REFERENCES cameras(id) ON DELETE CASCADE,
    nome        VARCHAR(128) NOT NULL,
    tipo        VARCHAR(32) NOT NULL DEFAULT 'restricted', -- restricted | counting
    -- poligono normalizzato (0..1) salvato come JSON: [[x,y],[x,y],...]
    poligono    JSONB NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ---------------------------------------------------------------
-- EVENTS
-- ---------------------------------------------------------------
CREATE TABLE IF NOT EXISTS events (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    camera_id   UUID NOT NULL REFERENCES cameras(id) ON DELETE CASCADE,
    zone_id     UUID REFERENCES zones(id) ON DELETE SET NULL,
    tipo_evento VARCHAR(32) NOT NULL,   -- person_detected | zone_alert | anomaly
    severity    VARCHAR(16) NOT NULL DEFAULT 'info', -- info | warning | critical
    track_id    VARCHAR(64),            -- id temporaneo assegnato dal tracker
    confidence  NUMERIC(4,3) NOT NULL,
    bbox_x      NUMERIC,
    bbox_y      NUMERIC,
    bbox_w      NUMERIC,
    bbox_h      NUMERIC,
    metadata    JSONB,                  -- campo libero per dati aggiuntivi
    "timestamp" TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_events_camera_ts ON events (camera_id, "timestamp" DESC);
CREATE INDEX IF NOT EXISTS idx_events_severity ON events (severity);

-- ---------------------------------------------------------------
-- SEED DI ESEMPIO
-- ---------------------------------------------------------------
INSERT INTO cameras (codice, nome, posizione, stato)
VALUES
    ('camera_01', 'Ingresso Principale', 'Piano Terra', 'offline'),
    ('camera_02', 'Area Server', 'Piano -1', 'offline')
ON CONFLICT (codice) DO NOTHING;

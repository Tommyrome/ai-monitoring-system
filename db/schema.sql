-- ============================================================
-- AI Camera Monitoring System - Database Schema (PostgreSQL)
-- ============================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- USERS
CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username      VARCHAR(64) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    ruolo         VARCHAR(32) NOT NULL DEFAULT 'operator',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- CAMERAS (solo una per la tua richiesta)
CREATE TABLE IF NOT EXISTS cameras (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    codice      VARCHAR(64) UNIQUE NOT NULL,
    nome        VARCHAR(128) NOT NULL,
    posizione   VARCHAR(128),
    stato       VARCHAR(16) NOT NULL DEFAULT 'offline',
    last_seen   TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ZONES (lasciato ma non più usato)
CREATE TABLE IF NOT EXISTS zones (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    camera_id   UUID NOT NULL REFERENCES cameras(id) ON DELETE CASCADE,
    nome        VARCHAR(128) NOT NULL,
    tipo        VARCHAR(32) NOT NULL DEFAULT 'restricted',
    poligono    JSONB NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- PERSONS - Nuova tabella per persone univoche e critiche
CREATE TABLE IF NOT EXISTS persons (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    track_id      VARCHAR(64) UNIQUE NOT NULL,        -- ID dal tracker
    nome          VARCHAR(128) DEFAULT 'Sconosciuto',
    is_critical   BOOLEAN DEFAULT FALSE,
    first_seen    TIMESTAMPTZ DEFAULT now(),
    last_seen     TIMESTAMPTZ DEFAULT now(),
    metadata      JSONB,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- EVENTS
CREATE TABLE IF NOT EXISTS events (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    camera_id   UUID NOT NULL REFERENCES cameras(id) ON DELETE CASCADE,
    person_id   UUID REFERENCES persons(id) ON DELETE SET NULL,  -- link opzionale
    tipo_evento VARCHAR(32) NOT NULL,
    severity    VARCHAR(16) NOT NULL DEFAULT 'info',
    track_id    VARCHAR(64),
    confidence  NUMERIC(4,3) NOT NULL,
    bbox_x      NUMERIC,
    bbox_y      NUMERIC,
    bbox_w      NUMERIC,
    bbox_h      NUMERIC,
    metadata    JSONB,
    "timestamp" TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_events_camera_ts ON events (camera_id, "timestamp" DESC);
CREATE INDEX IF NOT EXISTS idx_events_severity ON events (severity);
CREATE INDEX IF NOT EXISTS idx_persons_critical ON persons (is_critical);

-- SEED (solo 1 camera)
INSERT INTO cameras (codice, nome, posizione, stato)
VALUES ('camera_01', 'Telecamera Principale', 'Ingresso', 'offline')
ON CONFLICT (codice) DO NOTHING;



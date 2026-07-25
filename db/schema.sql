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
-- tipo_evento e' sempre NORMAL o CRITICAL: e' il backend a deciderlo in base
-- allo stato is_critical della persona riconosciuta al momento dell'evento
-- (unica fonte di verita', cosi' un cambio di stato si riflette subito).
CREATE TABLE IF NOT EXISTS events (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    camera_id   UUID NOT NULL REFERENCES cameras(id) ON DELETE CASCADE,
    person_id   UUID REFERENCES persons(id) ON DELETE SET NULL,
    tipo_evento VARCHAR(16) NOT NULL CHECK (tipo_evento IN ('NORMAL', 'CRITICAL')),
    track_id    VARCHAR(64),
    confidence  NUMERIC(4,3),
    bbox_x      NUMERIC,
    bbox_y      NUMERIC,
    bbox_w      NUMERIC,
    bbox_h      NUMERIC,
    metadata    JSONB,
    "timestamp" TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_events_camera_ts ON events (camera_id, "timestamp" DESC);
CREATE INDEX IF NOT EXISTS idx_events_tipo ON events (tipo_evento);
CREATE INDEX IF NOT EXISTS idx_events_person ON events (person_id);
CREATE INDEX IF NOT EXISTS idx_persons_critical ON persons (is_critical);
CREATE INDEX IF NOT EXISTS idx_persons_track_id ON persons (track_id);

-- SEED (solo 1 camera)
INSERT INTO cameras (codice, nome, posizione, stato)
VALUES ('camera_01', 'Telecamera Principale', 'Ingresso', 'offline')
ON CONFLICT (codice) DO NOTHING;



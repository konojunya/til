-- Matching Service内の異なるイベント種類を、1つの汎用Outboxで管理する。
CREATE TABLE outbox_events (
    id TEXT COLLATE "C" NOT NULL,
    event_type TEXT COLLATE "C" NOT NULL,
    payload JSONB NOT NULL,
    occurred_at timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz,

    CONSTRAINT outbox_events_pkey PRIMARY KEY (id),
    CONSTRAINT outbox_events_id_not_empty CHECK (id <> ''),
    CONSTRAINT outbox_events_type_not_empty CHECK (event_type <> ''),
    CONSTRAINT outbox_events_payload_object
        CHECK (jsonb_typeof(payload) = 'object')
);

-- 未配信イベントだけを発生順に取得するPublisherのqueryを支える。
CREATE INDEX outbox_events_pending_idx
    ON outbox_events (occurred_at, id)
    WHERE published_at IS NULL;
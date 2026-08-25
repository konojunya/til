-- UserIDは表示名ではなく識別子なので、自然言語順ではなくバイト順で比較する。
-- COLLATE "C"により、PostgreSQLの比較順をGoのstring比較と揃える。
CREATE TABLE users (
    id TEXT COLLATE "C" NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT users_pkey PRIMARY KEY (id),
    CONSTRAINT users_id_not_empty CHECK (id <> '')
);

-- Likeは方向を持つため、(sender_id, receiver_id)の列順を区別して一意にする。
CREATE TABLE likes (
    sender_id TEXT COLLATE "C" NOT NULL,
    receiver_id TEXT COLLATE "C" NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT likes_pkey PRIMARY KEY (sender_id, receiver_id),
    CONSTRAINT likes_not_self CHECK (sender_id <> receiver_id),
    CONSTRAINT likes_sender_fk FOREIGN KEY (sender_id) REFERENCES users (id),
    CONSTRAINT likes_receiver_fk FOREIGN KEY (receiver_id) REFERENCES users (id)
);

-- Matchは方向を持たないため、常に小さいIDをuser_low_idへ保存する。
-- user_low_id < user_high_idは自己Matchと逆順Matchを同時に拒否する。
CREATE TABLE matches (
    user_low_id TEXT COLLATE "C" NOT NULL,
    user_high_id TEXT COLLATE "C" NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT matches_pkey PRIMARY KEY (user_low_id, user_high_id),
    CONSTRAINT matches_users_ordered CHECK (user_low_id < user_high_id),
    CONSTRAINT matches_user_low_fk FOREIGN KEY (user_low_id) REFERENCES users (id),
    CONSTRAINT matches_user_high_fk FOREIGN KEY (user_high_id) REFERENCES users (id)
);

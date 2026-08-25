-- name: SaveLike :execrows
INSERT INTO likes (
    sender_id,
    receiver_id
) VALUES (
    $1,
    $2
)
ON CONFLICT (sender_id, receiver_id) DO NOTHING;

-- EXISTSは対象が0件ならfalse、1件以上ならtrueというboolean 1行を必ず返す。
-- name: HasLike :one
SELECT EXISTS (
    SELECT 1
    FROM likes
    WHERE sender_id = $1
      AND receiver_id = $2
);

-- name: SaveMatch :execrows
INSERT INTO matches (
    user_low_id,
    user_high_id
) VALUES (
    $1,
    $2
)
ON CONFLICT (user_low_id, user_high_id) DO NOTHING;

-- name: LockMatchingPair :exec
SELECT pg_advisory_xact_lock(
    hashtext(
        LEAST(
            sqlc.arg(first_user_id)::text,
            sqlc.arg(second_user_id)::text
        )
    ),
    hashtext(
        GREATEST(
            sqlc.arg(first_user_id)::text,
            sqlc.arg(second_user_id)::text
        )
    )
);
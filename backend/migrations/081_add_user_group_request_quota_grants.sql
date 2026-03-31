CREATE TABLE IF NOT EXISTS user_group_request_quota_grants (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    redeem_code_id BIGINT REFERENCES redeem_codes(id) ON DELETE SET NULL,
    request_quota_total BIGINT NOT NULL CHECK (request_quota_total > 0),
    request_quota_used BIGINT NOT NULL DEFAULT 0 CHECK (request_quota_used >= 0),
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_user_group_request_quota_grants_redeem_code_id
    ON user_group_request_quota_grants (redeem_code_id)
    WHERE redeem_code_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_user_group_request_quota_grants_user_group_expires
    ON user_group_request_quota_grants (user_id, group_id, expires_at, id);

UPDATE redeem_codes
SET validity_days = 30
WHERE type = 'group_request_quota'
  AND COALESCE(validity_days, 0) <= 0;

CREATE TEMP TABLE IF NOT EXISTS tmp_user_group_request_quota_grant_backfill (
    redeem_code_id BIGINT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    group_id BIGINT NOT NULL,
    request_quota_total BIGINT NOT NULL,
    request_quota_used BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
) ON COMMIT DROP;

TRUNCATE TABLE tmp_user_group_request_quota_grant_backfill;

INSERT INTO tmp_user_group_request_quota_grant_backfill (
    redeem_code_id,
    user_id,
    group_id,
    request_quota_total,
    request_quota_used,
    created_at,
    expires_at
)
SELECT
    rc.id,
    rc.used_by,
    rc.group_id,
    GREATEST(rc.value::BIGINT, 0),
    0,
    COALESCE(rc.used_at, rc.created_at),
    COALESCE(rc.used_at, rc.created_at) + make_interval(days => GREATEST(COALESCE(rc.validity_days, 30), 1))
FROM redeem_codes rc
WHERE rc.type = 'group_request_quota'
  AND rc.status = 'used'
  AND rc.used_by IS NOT NULL
  AND rc.group_id IS NOT NULL
  AND NOT EXISTS (
      SELECT 1
      FROM user_group_request_quota_grants g
      WHERE g.redeem_code_id = rc.id
  );

DO $$
DECLARE
    quota_group RECORD;
    grant_row RECORD;
    current_total BIGINT;
    current_used BIGINT;
    current_remaining BIGINT;
    active_recent_remaining BIGINT;
    temp_total BIGINT;
    temp_used BIGINT;
    remaining_used BIGINT;
    remaining_cap BIGINT;
    grant_available BIGINT;
    consume BIGINT;
BEGIN
    FOR quota_group IN
        SELECT user_id, group_id
        FROM tmp_user_group_request_quota_grant_backfill
        GROUP BY user_id, group_id
    LOOP
        SELECT request_quota, request_quota_used
        INTO current_total, current_used
        FROM user_group_request_quotas
        WHERE user_id = quota_group.user_id
          AND group_id = quota_group.group_id;

        current_total := COALESCE(current_total, 0);
        current_used := COALESCE(current_used, 0);
        current_remaining := GREATEST(current_total - current_used, 0);
        remaining_used := current_used;

        FOR grant_row IN
            SELECT redeem_code_id, request_quota_total
            FROM tmp_user_group_request_quota_grant_backfill
            WHERE user_id = quota_group.user_id
              AND group_id = quota_group.group_id
            ORDER BY created_at ASC, redeem_code_id ASC
        LOOP
            EXIT WHEN remaining_used <= 0;
            consume := LEAST(remaining_used, grant_row.request_quota_total);
            UPDATE tmp_user_group_request_quota_grant_backfill
            SET request_quota_used = consume
            WHERE redeem_code_id = grant_row.redeem_code_id;
            remaining_used := remaining_used - consume;
        END LOOP;

        SELECT COALESCE(SUM(request_quota_total - request_quota_used), 0)
        INTO active_recent_remaining
        FROM tmp_user_group_request_quota_grant_backfill
        WHERE user_id = quota_group.user_id
          AND group_id = quota_group.group_id
          AND expires_at > NOW();

        remaining_cap := GREATEST(active_recent_remaining - current_remaining, 0);
        IF remaining_cap > 0 THEN
            FOR grant_row IN
                SELECT redeem_code_id, request_quota_total, request_quota_used
                FROM tmp_user_group_request_quota_grant_backfill
                WHERE user_id = quota_group.user_id
                  AND group_id = quota_group.group_id
                  AND expires_at > NOW()
                ORDER BY expires_at ASC, redeem_code_id ASC
            LOOP
                EXIT WHEN remaining_cap <= 0;
                grant_available := grant_row.request_quota_total - grant_row.request_quota_used;
                IF grant_available <= 0 THEN
                    CONTINUE;
                END IF;
                consume := LEAST(remaining_cap, grant_available);
                UPDATE tmp_user_group_request_quota_grant_backfill
                SET request_quota_used = request_quota_used + consume
                WHERE redeem_code_id = grant_row.redeem_code_id;
                remaining_cap := remaining_cap - consume;
            END LOOP;
        END IF;

        SELECT
            COALESCE(SUM(request_quota_total), 0),
            COALESCE(SUM(request_quota_used), 0)
        INTO temp_total, temp_used
        FROM tmp_user_group_request_quota_grant_backfill
        WHERE user_id = quota_group.user_id
          AND group_id = quota_group.group_id;

        UPDATE user_group_request_quotas
        SET
            request_quota = GREATEST(request_quota - temp_total, 0),
            request_quota_used = GREATEST(request_quota_used - temp_used, 0),
            updated_at = NOW()
        WHERE user_id = quota_group.user_id
          AND group_id = quota_group.group_id;
    END LOOP;
END $$;

INSERT INTO user_group_request_quota_grants (
    redeem_code_id,
    user_id,
    group_id,
    request_quota_total,
    request_quota_used,
    expires_at,
    created_at,
    updated_at
)
SELECT
    redeem_code_id,
    user_id,
    group_id,
    request_quota_total,
    request_quota_used,
    expires_at,
    created_at,
    NOW()
FROM tmp_user_group_request_quota_grant_backfill
ON CONFLICT DO NOTHING;

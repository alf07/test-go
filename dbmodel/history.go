package dbmodel

const QueryAddHistory = `
    INSERT INTO history (
        wallet_id,
        amount,
        operation,
        created_at,
        status
    )
    VALUES ($1, $2, $3, CURRENT_TIMESTAMP, 'COMPLETED');
`
const queryUpdateHistory = `
    UPDATE history
    SET status = 'PROCESSING',
        attempts = attempts + 1,
        last_attempt_at = CURRENT_TIMESTAMP,
        error_message = NULL
    WHERE id = $1
      AND status IN ('PENDING', 'FAILED')
    RETURNING id, wallet_id, amount, operation, status, attempts;
`

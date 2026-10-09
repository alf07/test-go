package dbmodel

const QueryGetWallet = `
    SELECT id, wallet_uuid, balance
    FROM wallet
    WHERE wallet_uuid = $1;
`

const QueryDepositBalance = `
    UPDATE wallet
    SET balance = balance + $1
    WHERE wallet_uuid = $2
    RETURNING id, wallet_uuid, balance;
`

const QueryWithdrawBalance = `
    UPDATE wallet
    SET balance = balance - $1
    WHERE wallet_uuid = $2
      AND balance >= $1
    RETURNING id, wallet_uuid, balance;
`
const QueryGetWalletID = `
    SELECT id
    FROM wallet
    WHERE wallet_uuid = $1;
`

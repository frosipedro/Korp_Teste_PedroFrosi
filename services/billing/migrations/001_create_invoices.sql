CREATE TYPE invoice_status AS ENUM ('open', 'closed');

CREATE TABLE IF NOT EXISTS invoices (
    id              SERIAL PRIMARY KEY,
    number          INTEGER         NOT NULL UNIQUE,
    status          invoice_status  NOT NULL DEFAULT 'open',
    closed_at       TIMESTAMPTZ,
    idempotency_key VARCHAR(100)    UNIQUE,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_invoices_number         ON invoices(number);
CREATE INDEX IF NOT EXISTS idx_invoices_status         ON invoices(status);
CREATE INDEX IF NOT EXISTS idx_invoices_idempotency    ON invoices(idempotency_key);

CREATE TABLE IF NOT EXISTS invoice_items (
    id          SERIAL PRIMARY KEY,
    invoice_id  INTEGER         NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    product_id  INTEGER         NOT NULL,
    product_code VARCHAR(50)    NOT NULL,
    description VARCHAR(255)    NOT NULL,
    quantity    INTEGER         NOT NULL CHECK (quantity > 0),
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_invoice_items_invoice_id ON invoice_items(invoice_id);

CREATE SEQUENCE IF NOT EXISTS invoice_number_seq START 1000;

CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_invoices_updated_at
    BEFORE UPDATE ON invoices
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();
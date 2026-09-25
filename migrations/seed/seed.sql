-- Demo data, loaded by hand: migrations run wherever the schema is deployed, production included.
-- Country code 999 is unassigned under ITU-T E.164, so these numbers cannot reach a subscriber.

INSERT INTO templates (type, body) VALUES
('reminder', E'{credit_number}\n\nDear {full_name},\n\nplease repay the amount of {amount} by {due_date}.\n\nKind regards\nACME'),
('dunning', E'{credit_number}\n\nDear {full_name},\n\n{amount} was due on {due_date} and is still outstanding. Please pay without further delay.\n\nACME'),
('termination', E'{credit_number}\n\nDear {full_name},\n\n{amount} has remained unpaid since {due_date}. We are terminating your loan agreement.\n\nACME')
ON CONFLICT (type) DO NOTHING;

INSERT INTO customers (credit_number, phone_number, full_name, amount_minor, currency, due_date) VALUES
('1000000001', '+999000000001', 'John Doe', 125000, 'EUR', '2026-09-25'),
('1000000002', '+999000000002', 'Jane Doe', 48050, 'EUR', '2026-08-30')
ON CONFLICT (credit_number) DO NOTHING;

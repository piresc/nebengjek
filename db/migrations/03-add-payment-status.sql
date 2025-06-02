-- Add status column to payments table
ALTER TABLE IF EXISTS payments ADD COLUMN IF NOT EXISTS status VARCHAR(20) DEFAULT 'PENDING';

-- Update existing records to have a default status of PROCESSED
UPDATE payments SET status = 'PROCESSED' WHERE status IS NULL;

-- Make status NOT NULL after updating existing records
ALTER TABLE payments ALTER COLUMN status SET NOT NULL;

-- Add check constraint to ensure status is one of the allowed values
ALTER TABLE payments ADD CONSTRAINT check_payment_status 
CHECK (status IN ('PENDING', 'ACCEPTED', 'REJECTED', 'PROCESSED'));

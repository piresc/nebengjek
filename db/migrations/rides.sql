-- Create enum type for ride status
CREATE TYPE ride_status AS ENUM ('pending', 'ongoing', 'completed');

-- Create rides table
CREATE TABLE IF NOT EXISTS rides (
    ride_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    driver_id UUID NOT NULL,
    customer_id UUID NOT NULL,
    status ride_status NOT NULL DEFAULT 'pending',
    total_cost INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (driver_id) REFERENCES users(id),
    FOREIGN KEY (customer_id) REFERENCES users(id)
);

-- Create billing_ledger table
CREATE TABLE IF NOT EXISTS billing_ledger (
    entry_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ride_id UUID NOT NULL,
    distance FLOAT NOT NULL,
    cost INT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (ride_id) REFERENCES rides(ride_id),
    CONSTRAINT positive_distance CHECK (distance > 0),
    CONSTRAINT positive_cost CHECK (cost > 0)
);

-- Create payments table
CREATE TABLE IF NOT EXISTS payments (
    payment_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ride_id UUID NOT NULL UNIQUE,
    adjusted_cost INT NOT NULL,
    admin_fee INT NOT NULL,
    driver_payout INT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (ride_id) REFERENCES rides(ride_id),
    CONSTRAINT positive_adjusted_cost CHECK (adjusted_cost > 0),
    CONSTRAINT positive_admin_fee CHECK (admin_fee > 0),
    CONSTRAINT positive_driver_payout CHECK (driver_payout > 0)
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_rides_driver_id ON rides(driver_id);
CREATE INDEX IF NOT EXISTS idx_rides_customer_id ON rides(customer_id);
CREATE INDEX IF NOT EXISTS idx_billing_ledger_ride_id ON billing_ledger(ride_id);
CREATE INDEX IF NOT EXISTS idx_payments_ride_id ON payments(ride_id);

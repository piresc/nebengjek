-- Users table
CREATE TABLE IF NOT EXISTS users (
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    msisdn character varying(20) NOT NULL,
    fullname character varying(255) NOT NULL,
    role character varying(20) NOT NULL, -- 'driver' or 'passenger'
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    is_active boolean NOT NULL DEFAULT true,
    CONSTRAINT users_pkey PRIMARY KEY (id),
    CONSTRAINT users_msisdn_key UNIQUE (msisdn)
);

-- Drivers table (additional info for users who are drivers)
CREATE TABLE IF NOT EXISTS drivers (
    user_id uuid NOT NULL,
    vehicle_type character varying(50) NOT NULL,
    vehicle_plate character varying(20) NOT NULL,
    CONSTRAINT drivers_pkey PRIMARY KEY (user_id),
    CONSTRAINT drivers_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Matches table
CREATE TABLE IF NOT EXISTS matches (
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    driver_id uuid NOT NULL,
    passenger_id uuid NOT NULL,
    driver_location point NOT NULL,
    passenger_location point NOT NULL,
    status match_status NOT NULL DEFAULT 'PENDING'::match_status,
    created_at timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    driver_confirmed boolean NOT NULL DEFAULT false,
    passenger_confirmed boolean NOT NULL DEFAULT false,
    target_location point NULL,
    CONSTRAINT matches_pkey PRIMARY KEY (id),
    CONSTRAINT matches_driver_id_fkey FOREIGN KEY (driver_id) REFERENCES users(id),
    CONSTRAINT matches_passenger_id_fkey FOREIGN KEY (passenger_id) REFERENCES users(id)
);

-- Rides table
CREATE TABLE IF NOT EXISTS rides (
    ride_id uuid NOT NULL DEFAULT gen_random_uuid(),
    driver_id uuid NOT NULL,
    passenger_id uuid NOT NULL,
    status ride_status NOT NULL DEFAULT 'PENDING'::ride_status,
    total_cost integer NOT NULL DEFAULT 0,
    created_at timestamp with time zone NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT rides_pkey PRIMARY KEY (ride_id),
    CONSTRAINT rides_driver_id_fkey FOREIGN KEY (driver_id) REFERENCES users(id),
    CONSTRAINT rides_passenger_id_fkey FOREIGN KEY (passenger_id) REFERENCES users(id)
);

-- Billing ledger table
CREATE TABLE IF NOT EXISTS billing_ledger (
    entry_id uuid NOT NULL DEFAULT gen_random_uuid(),
    ride_id uuid NOT NULL,
    distance double precision NOT NULL,
    cost integer NOT NULL,
    created_at timestamp with time zone NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT billing_ledger_pkey PRIMARY KEY (entry_id),
    CONSTRAINT billing_ledger_ride_id_fkey FOREIGN KEY (ride_id) REFERENCES rides(ride_id),
    CONSTRAINT positive_distance CHECK (distance > 0),
    CONSTRAINT positive_cost CHECK (cost > 0)
);

-- Payments table
CREATE TABLE IF NOT EXISTS payments (
    payment_id uuid NOT NULL DEFAULT gen_random_uuid(),
    ride_id uuid NOT NULL,
    adjusted_cost integer NOT NULL,
    admin_fee integer NOT NULL,
    driver_payout integer NOT NULL,
    created_at timestamp with time zone NULL DEFAULT CURRENT_TIMESTAMP,
    status character varying(20) NOT NULL DEFAULT 'PENDING'::character varying,
    CONSTRAINT payments_pkey PRIMARY KEY (payment_id),
    CONSTRAINT payments_ride_id_fkey FOREIGN KEY (ride_id) REFERENCES rides(ride_id),
    CONSTRAINT payments_ride_id_key UNIQUE (ride_id),
    CONSTRAINT positive_adjusted_cost CHECK (adjusted_cost > 0),
    CONSTRAINT positive_admin_fee CHECK (admin_fee > 0),
    CONSTRAINT positive_driver_payout CHECK (driver_payout > 0),
    CONSTRAINT check_payment_status CHECK (status IN ('PENDING', 'ACCEPTED', 'REJECTED', 'PROCESSED'))
);

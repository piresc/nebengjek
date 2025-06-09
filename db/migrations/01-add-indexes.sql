-- Matches table indexes
CREATE INDEX idx_matches_driver_id ON matches(driver_id);
CREATE INDEX idx_matches_passenger_id ON matches(passenger_id);
CREATE INDEX idx_matches_driver_location ON matches USING GIST (driver_location);
CREATE INDEX idx_matches_passenger_location ON matches USING GIST (passenger_location);
CREATE INDEX idx_matches_driver_confirmed ON matches(driver_confirmed);
CREATE INDEX idx_matches_passenger_confirmed ON matches(passenger_confirmed);
CREATE INDEX idx_matches_status_confirmations ON matches(status, driver_confirmed, passenger_confirmed);

-- Rides table indexes
CREATE INDEX idx_rides_driver_id ON rides(driver_id);
CREATE INDEX idx_rides_passenger_id ON rides(passenger_id);

-- Billing ledger indexes
CREATE INDEX idx_billing_ledger_ride_id ON billing_ledger(ride_id);

-- Payments table indexes
CREATE INDEX idx_payments_ride_id ON payments(ride_id);

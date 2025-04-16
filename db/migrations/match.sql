CREATE TYPE match_status AS ENUM ('PENDING', 'ACCEPTED', 'REJECTED');

CREATE TABLE IF NOT EXISTS matches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    driver_id UUID NOT NULL,
    passenger_id UUID NOT NULL,
    driver_location point NOT NULL,
    passenger_location point NOT NULL,
    status match_status NOT NULL DEFAULT 'PENDING',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (driver_id) REFERENCES users(id),
    FOREIGN KEY (passenger_id) REFERENCES users(id)
);

-- Create an index on driver_id and passenger_id for faster lookups
CREATE INDEX idx_matches_driver_id ON matches(driver_id);
CREATE INDEX idx_matches_passenger_id ON matches(passenger_id);

-- Create a spatial index on location columns for faster geographical queries
CREATE INDEX idx_matches_driver_location ON matches USING GIST (driver_location);
CREATE INDEX idx_matches_passenger_location ON matches USING GIST (passenger_location);

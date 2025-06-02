-- Add target_location to matches table
ALTER TABLE matches ADD COLUMN target_location POINT;


-- Update the match_status enum to include new confirmation states
ALTER TYPE ride_status ADD VALUE 'PENDING';
ALTER TYPE ride_status ADD VALUE 'PICKUP';
ALTER TYPE ride_status ADD VALUE 'ONGOING';
ALTER TYPE ride_status ADD VALUE 'COMPLETED';
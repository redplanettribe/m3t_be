ALTER TABLE rooms
    DROP COLUMN IF EXISTS capacity,
    DROP COLUMN IF EXISTS description,
    DROP COLUMN IF EXISTS how_to_get_there;

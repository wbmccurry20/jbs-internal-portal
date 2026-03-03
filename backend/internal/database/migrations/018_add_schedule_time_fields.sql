-- Add actual start/end time fields to training schedule items.
-- Replaces the restrictive morning/afternoon/all-day time_slot enum
-- with flexible time inputs (e.g. "09:00", "14:30").

-- Drop the old CHECK constraint so time_slot is no longer restricted
ALTER TABLE training_schedule_items
    DROP CONSTRAINT IF EXISTS valid_time_slot;

-- Add time fields (HH:MM in 24-hour format, nullable = "all day")
ALTER TABLE training_schedule_items
    ADD COLUMN IF NOT EXISTS start_time VARCHAR(5),
    ADD COLUMN IF NOT EXISTS end_time   VARCHAR(5);

-- Backfill sensible defaults for existing rows that had a time_slot set
UPDATE training_schedule_items SET start_time = '08:00', end_time = '12:00' WHERE time_slot = 'morning'  AND start_time IS NULL;
UPDATE training_schedule_items SET start_time = '13:00', end_time = '17:00' WHERE time_slot = 'afternoon' AND start_time IS NULL;

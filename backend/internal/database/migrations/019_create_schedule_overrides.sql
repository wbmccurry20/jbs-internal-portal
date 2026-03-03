-- Per-trainee schedule overrides.
-- Allows admins to customise an individual trainee's schedule
-- by adding, removing, or replacing items from the base program.

CREATE TABLE IF NOT EXISTS assignment_schedule_overrides (
    id         SERIAL PRIMARY KEY,
    assignment_id INTEGER NOT NULL REFERENCES trainee_assignments(id) ON DELETE CASCADE,

    -- Which slot this override targets
    week_number INTEGER NOT NULL CHECK (week_number >= 1),
    day_of_week INTEGER NOT NULL CHECK (day_of_week >= 1 AND day_of_week <= 5),

    -- Action: 'add' inserts a new item, 'remove' hides a base-program item, 'replace' swaps it
    action      VARCHAR(10) NOT NULL DEFAULT 'add',

    -- For 'remove' actions, reference the base item to hide
    base_item_id INTEGER REFERENCES training_schedule_items(id) ON DELETE CASCADE,

    -- Content fields (used by 'add' and 'replace')
    title       VARCHAR(500),
    description TEXT,
    day_title   VARCHAR(255),
    link_url    TEXT,
    link_label  VARCHAR(255),
    time_slot   VARCHAR(50) DEFAULT 'all-day',
    start_time  VARCHAR(5),
    end_time    VARCHAR(5),
    sort_order  INTEGER NOT NULL DEFAULT 0,
    is_highlight BOOLEAN NOT NULL DEFAULT false,

    created_by  INTEGER REFERENCES users(id),
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT valid_override_action CHECK (action IN ('add', 'remove', 'replace'))
);

CREATE INDEX IF NOT EXISTS idx_overrides_assignment ON assignment_schedule_overrides(assignment_id, week_number, day_of_week);

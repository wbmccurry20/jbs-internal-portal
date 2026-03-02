-- Training Portal Tables
-- Programs are templates (e.g. "APM 30-Day Onboarding")
-- Schedule items define daily activities within a program
-- Assignments link a trainee to a program with a start date

-- Training programs (templates)
CREATE TABLE IF NOT EXISTS training_programs (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    duration_weeks INTEGER NOT NULL DEFAULT 4,
    created_by INTEGER REFERENCES users(id),
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Schedule items for each day of a program
CREATE TABLE IF NOT EXISTS training_schedule_items (
    id SERIAL PRIMARY KEY,
    program_id INTEGER NOT NULL REFERENCES training_programs(id) ON DELETE CASCADE,
    week_number INTEGER NOT NULL CHECK (week_number >= 1),
    day_of_week INTEGER NOT NULL CHECK (day_of_week >= 1 AND day_of_week <= 5),
    day_title VARCHAR(255),
    title VARCHAR(500) NOT NULL,
    description TEXT,
    link_url TEXT,
    link_label VARCHAR(255),
    time_slot VARCHAR(50) DEFAULT 'all-day',
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_highlight BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT valid_time_slot CHECK (time_slot IN ('morning', 'afternoon', 'all-day'))
);

-- Trainee assignments (link user to a program)
CREATE TABLE IF NOT EXISTS trainee_assignments (
    id SERIAL PRIMARY KEY,
    program_id INTEGER NOT NULL REFERENCES training_programs(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    start_date DATE NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    notes TEXT,
    assigned_by INTEGER REFERENCES users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT valid_assignment_status CHECK (status IN ('active', 'completed', 'paused')),
    CONSTRAINT unique_trainee_program UNIQUE (program_id, user_id)
);

-- Training resources (general links/docs for a program)
CREATE TABLE IF NOT EXISTS training_resources (
    id SERIAL PRIMARY KEY,
    program_id INTEGER NOT NULL REFERENCES training_programs(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    url TEXT NOT NULL,
    category VARCHAR(100) DEFAULT 'General',
    description TEXT,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_training_schedule_program ON training_schedule_items(program_id, week_number, day_of_week);
CREATE INDEX IF NOT EXISTS idx_trainee_assignments_user ON trainee_assignments(user_id, status);
CREATE INDEX IF NOT EXISTS idx_trainee_assignments_program ON trainee_assignments(program_id);
CREATE INDEX IF NOT EXISTS idx_training_resources_program ON training_resources(program_id);

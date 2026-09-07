CREATE TABLE testimonials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    quote TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_testimonials_user_id ON testimonials(user_id);
CREATE INDEX idx_testimonials_status ON testimonials(status);

CREATE TRIGGER testimonials_set_updated_at
    BEFORE UPDATE ON testimonials
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- Ensure one testimonial per user
CREATE UNIQUE INDEX idx_testimonials_user_one ON testimonials(user_id) WHERE status IN ('pending', 'approved');
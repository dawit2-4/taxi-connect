
CREATE TABLE queue_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    from_terminal_id UUID NOT NULL,
    to_terminal_id UUID NOT NULL,
    reported_wait_minutes INTEGER NULL,
    reported_queue_length INTEGER NULL,
    reporter_notes TEXT NULL,
    reported_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_queue_reports_from_terminal
        FOREIGN KEY (from_terminal_id)
        REFERENCES terminals(id)
        ON DELETE RESTRICT,

    CONSTRAINT fk_queue_reports_to_terminal
        FOREIGN KEY (to_terminal_id)
        REFERENCES terminals(id)
        ON DELETE RESTRICT
);

CREATE INDEX idx_queue_reports_route_on_queue_reports
    ON queue_reports(
        from_terminal_id,
        to_terminal_id,
        reported_at DESC
    );

CREATE INDEX idx_queue_reports_reported_at_on_queue_reports
    ON queue_reports(reported_at);

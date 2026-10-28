-- Seed terminals with approximate Addis Ababa coordinates
INSERT INTO terminals (name, location) VALUES
('Megenagna', 'POINT(38.7878 9.0375)'),
('Garment', 'POINT(38.7578 8.9800)'),
('Bole', 'POINT(38.8000 8.9667)'),
('Mexico', 'POINT(38.7667 8.9500)'),
('Saris Abo', 'POINT(38.7833 9.0167)')
ON CONFLICT (name) DO NOTHING;

-- Seed route edges (directional connections)
INSERT INTO route_edges (from_terminal_id, to_terminal_id, travel_time_minutes)
SELECT
    t1.id, t2.id,
    CASE
        WHEN t1.name = 'Megenagna' AND t2.name = 'Garment' THEN 15
        WHEN t1.name = 'Megenagna' AND t2.name = 'Bole' THEN 10
        WHEN t1.name = 'Bole' AND t2.name = 'Garment' THEN 12
        WHEN t1.name = 'Megenagna' AND t2.name = 'Mexico' THEN 20
        WHEN t1.name = 'Mexico' AND t2.name = 'Garment' THEN 18
        WHEN t1.name = 'Megenagna' AND t2.name = 'Saris Abo' THEN 14
        WHEN t1.name = 'Saris Abo' AND t2.name = 'Garment' THEN 16
        ELSE 10 -- default
    END
FROM terminals t1, terminals t2
WHERE
    (t1.name = 'Megenagna' AND t2.name IN ('Garment', 'Bole', 'Mexico', 'Saris Abo')) OR
    (t1.name = 'Bole' AND t2.name = 'Garment') OR
    (t1.name = 'Mexico' AND t2.name = 'Garment') OR
    (t1.name = 'Saris Abo' AND t2.name = 'Garment')
ON CONFLICT (from_terminal_id, to_terminal_id) DO NOTHING;
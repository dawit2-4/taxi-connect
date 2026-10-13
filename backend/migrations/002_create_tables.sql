 -- Terminals table
  CREATE TABLE terminals (
      id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      name VARCHAR(100) NOT NULL UNIQUE,
      location GEOGRAPHY(Point, 4326) NOT NULL,
      created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
      updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
  );

  -- Directional route edges table
  CREATE TABLE route_edges (
      id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      from_terminal_id UUID NOT NULL REFERENCES terminals(id) ON DELETE RESTRICT,
      to_terminal_id UUID NOT NULL REFERENCES terminals(id) ON DELETE RESTRICT,
      travel_time_minutes INTEGER NOT NULL CHECK (travel_time_minutes >= 0),
      created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
      updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
      UNIQUE (from_terminal_id, to_terminal_id)
  );

  -- Indexes for foreign keys (optional but helpful for query performance)
  CREATE INDEX idx_route_edges_from ON route_edges(from_terminal_id);
  CREATE INDEX idx_route_edges_to ON route_edges(to_terminal_id);
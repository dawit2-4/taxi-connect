-- Enable pgcrypto for UUID generation
  CREATE EXTENSION IF NOT EXISTS pgcrypto;

  -- Enable PostGIS for geographic data types
  CREATE EXTENSION IF NOT EXISTS postgis;
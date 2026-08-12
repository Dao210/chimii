CREATE TABLE part_definition (
    part_key TEXT NOT NULL,
    name TEXT NOT NULL,
    category TEXT NOT NULL,
    popularity_rank INTEGER NOT NULL CHECK (popularity_rank > 0),
    certification_level TEXT NOT NULL DEFAULT 'asset_only'
        CHECK (certification_level IN ('asset_only', 'basic', 'advanced', 'certified')),
    auto_build_eligible BOOLEAN NOT NULL DEFAULT FALSE,
    geometry_profile TEXT NOT NULL DEFAULT 'asset_only'
        CHECK (geometry_profile IN ('asset_only', 'stud_tube_rect', 'tile_rect', 'legacy_special')),
    studs_x INTEGER NOT NULL DEFAULT 0 CHECK (studs_x >= 0),
    studs_z INTEGER NOT NULL DEFAULT 0 CHECK (studs_z >= 0),
    plates_y INTEGER NOT NULL DEFAULT 0 CHECK (plates_y >= 0),
    default_quantity INTEGER NOT NULL DEFAULT 0 CHECK (default_quantity >= 0),
    has_top_studs BOOLEAN NOT NULL DEFAULT FALSE,
    has_bottom_receptors BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

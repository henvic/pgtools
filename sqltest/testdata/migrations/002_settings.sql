CREATE TYPE status_type AS ENUM ('active', 'inactive');

CREATE TABLE settings (
	id text PRIMARY KEY,
	name text NOT NULL,
	code text UNIQUE NOT NULL,
	status status_type NOT NULL DEFAULT 'inactive',
	style jsonb,
	created_at timestamp with time zone NOT NULL DEFAULT now(),
	modified_at timestamp with time zone NOT NULL DEFAULT now()
);
CREATE INDEX brand_code_idx ON settings(code);

---- create above / drop below ----

DROP TABLE IF EXISTS settings;
DROP TYPE status_type;

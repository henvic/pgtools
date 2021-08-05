-- Initial SQL schema for the media microservice.

CREATE TYPE media_type AS ENUM ('photo', 'illustration', 'sketch');

CREATE TABLE media (
	id text PRIMARY KEY,
	name text NOT NULL,
	source media_type NOT NULL,
	url text NOT NULL,
	created_at timestamp with time zone NOT NULL DEFAULT now(),
	modified_at timestamp with time zone NOT NULL DEFAULT now()
);

CREATE INDEX media_name ON media(name text_pattern_ops);

---- create above / drop below ----
DROP TABLE IF EXISTS media;
DROP TYPE IF EXISTS media_type;

CREATE TABLE posts (
	id text PRIMARY KEY,
	name text NOT NULL,
	message text NOT NULL, 
	created_at timestamp with time zone NOT NULL DEFAULT now(),
	modified_at timestamp with time zone NOT NULL DEFAULT now()
);
---- create above / drop below ----

DROP TABLE IF EXISTS posts;

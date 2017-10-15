-- CREATE DATABASE forum_db OWNER docker;

CREATE EXTENSION IF NOT EXISTS citext;

CREATE EXTENSION IF NOT EXISTS ltree;

CREATE TABLE IF NOT EXISTS "user"
(
  id       BIGSERIAL   NOT NULL
    CONSTRAINT user_pkey
    PRIMARY KEY,
  nickname CITEXT,
  fullname VARCHAR(64) NOT NULL,
  email    CITEXT NOT NULL,
  about    TEXT
);

CREATE UNIQUE INDEX IF NOT EXISTS user_nickname_uindex
  ON "user" (nickname);

CREATE UNIQUE INDEX IF NOT EXISTS user_email_uindex
  ON "user" (email);

-- auto-generated definition
CREATE TABLE IF NOT EXISTS forum
(
id       BIGSERIAL    NOT NULL
CONSTRAINT forum_db_pkey
PRIMARY KEY,
slug     CITEXT       NOT NULL,
title    VARCHAR(128) NOT NULL,
owner_id BIGINT
CONSTRAINT forum_user_id_fk
REFERENCES "user",
posts    BIGINT  DEFAULT 0,
threads  INTEGER DEFAULT 0
);

CREATE INDEX IF NOT EXISTS forum_owner_id_idx ON forum (owner_id);

CREATE UNIQUE INDEX IF NOT EXISTS forum_slug_uindex
ON forum (slug);

-- auto-generated definition
CREATE TABLE IF NOT EXISTS thread
(
id       SERIAL                                         NOT NULL
CONSTRAINT thread_pkey
PRIMARY KEY,
slug     CITEXT DEFAULT NULL :: CITEXT,
title    VARCHAR(128) DEFAULT NULL :: CHARACTER VARYING NOT NULL,
message  TEXT                                           NOT NULL,
forum_id BIGINT                                         NOT NULL
CONSTRAINT thread_forum_id_fk
REFERENCES forum,
owner_id INTEGER                                        NOT NULL
CONSTRAINT thread_user_id_fk
REFERENCES "user",
created  TIMESTAMP WITH TIME ZONE DEFAULT now()         NOT NULL,
votes    INTEGER DEFAULT 0                              NOT NULL
);


CREATE UNIQUE INDEX IF NOT EXISTS thread_slug_uindex
ON thread (lower(slug));

CREATE INDEX IF NOT EXISTS thread_forum_id_idx ON thread (forum_id);
CREATE INDEX IF NOT EXISTS thread_owner_id_idx ON thread (owner_id);

CREATE or REPLACE FUNCTION update_count_threads()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
  UPDATE forum
  SET threads = threads + 1 WHERE forum.id = new.forum_id;
  RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS count_threads_tgr ON thread;

CREATE TRIGGER count_threads_tgr
BEFORE INSERT
ON thread
FOR EACH ROW
EXECUTE PROCEDURE update_count_threads();

-- auto-generated definition
CREATE TABLE IF NOT EXISTS post
(
id        BIGSERIAL                              NOT NULL
CONSTRAINT post_pkey
PRIMARY KEY,
message   TEXT                                   NOT NULL,
thread_id INTEGER                                NOT NULL
CONSTRAINT post_thread_id_fk
REFERENCES thread,
parent    BIGINT DEFAULT 0                       NOT NULL,
owner_id  BIGINT                                 NOT NULL
CONSTRAINT post_user_id_fk
REFERENCES "user",
created   TIMESTAMP WITH TIME ZONE DEFAULT now() NOT NULL,
isedited  BOOLEAN DEFAULT FALSE                  NOT NULL,
path      LTREE
);

CREATE INDEX IF NOT EXISTS post_thread_id_idx
ON post (thread_id);

CREATE INDEX IF NOT EXISTS post_parent_id_idx
ON post (parent);

CREATE INDEX IF NOT EXISTS post_owner_id_idx
ON post (owner_id);

CREATE INDEX IF NOT EXISTS post_parent_path_idx
ON post USING GIST (path);

CREATE INDEX IF NOT EXISTS post_parent_id_idx
ON post (parent);

CREATE OR REPLACE FUNCTION update_section_parent_path()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
  p LTREE;
BEGIN
  IF NEW.parent = 0
  THEN
    NEW.path = new.id :: TEXT :: LTREE;
  ELSEIF TG_OP = 'INSERT'
  THEN
      SELECT pst.path || New.id :: TEXT
  FROM post pst
  WHERE id = NEW.parent
  INTO p;
      IF p IS NULL
  THEN
        RAISE EXCEPTION 'Invalid parent_id %', NEW.parent;
      END IF;
      NEW.path = p;
  END IF;
  RETURN NEW;
END;
$$;

CREATE TRIGGER parent_path_tgr
BEFORE INSERT
ON post
FOR EACH ROW
EXECUTE PROCEDURE update_section_parent_path();

-- CREATE OR REPLACE FUNCTION update_count_posts()
-- RETURNS TRIGGER
-- LANGUAGE plpgsql
-- AS $$
-- DECLARE IDD INT;
-- BEGIN
--   SELECT f.id
--   FROM thread t
--   JOIN forum f ON t.forum_id = f.id
--   WHERE t.id = new.thread_id
--   INTO IDD;
--   RAISE NOTICE '%', IDD;
--   UPDATE forum
--   SET posts = posts + 1
--   WHERE forum.id = IDD;
--   RETURN NEW;
-- END;
-- $$;

-- CREATE TRIGGER count_posts_tgr
-- BEFORE INSERT
-- ON post
-- FOR EACH ROW
-- EXECUTE PROCEDURE update_count_posts();


CREATE TABLE IF NOT EXISTS votes
(
  id        SERIAL  NOT NULL
    CONSTRAINT votes_pkey
    PRIMARY KEY,
  user_id   BIGINT  NOT NULL
    CONSTRAINT votes_user_id_fk
    REFERENCES "user",
  thread_id INTEGER NOT NULL
    CONSTRAINT votes_thread_id_fk
    REFERENCES thread,
  voice     INTEGER NOT NULL
);

CREATE UNIQUE INDEX votes_user_id_thread_id_uindex
  ON votes (user_id, thread_id);

CREATE OR REPLACE FUNCTION update_count_votes()
  RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
  IF (TG_OP = 'UPDATE')
  THEN
    IF new.voice != old.voice
    THEN
      UPDATE thread
      SET votes = votes + (2 * new.voice)
      WHERE thread.id = NEW.thread_id;
      RETURN NEW;
    END IF;
    RETURN NEW;
  ELSE
    UPDATE thread
    SET votes = votes + new.voice
    WHERE thread.id = NEW.thread_id;
    RETURN NEW;
  END IF;
END;
$$;

CREATE TRIGGER update_count_votes_trig
AFTER INSERT OR UPDATE
  ON votes
FOR EACH ROW
EXECUTE PROCEDURE update_count_votes();


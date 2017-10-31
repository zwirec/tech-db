-- CREATE DATABASE forum_db OWNER docker;

CREATE EXTENSION IF NOT EXISTS citext;

-- CREATE EXTENSION IF NOT EXISTS ltree;

CREATE TABLE IF NOT EXISTS "user"
(
  id       BIGSERIAL   NOT NULL
    CONSTRAINT user_pkey
    PRIMARY KEY,
  nickname CITEXT,
  fullname VARCHAR(64) NOT NULL,
  email    CITEXT      NOT NULL,
  about    TEXT
);

CREATE UNIQUE INDEX IF NOT EXISTS user_nickname_uindex
  ON "user" (nickname);

CREATE UNIQUE INDEX IF NOT EXISTS user_email_uindex
  ON "user" (email);

CREATE INDEX IF NOT EXISTS user_nickname_email_index
  ON "user" (nickname, email);

-- auto-generated definition
CREATE TABLE IF NOT EXISTS forum
(
  id             BIGSERIAL    NOT NULL
    CONSTRAINT forum_db_pkey
    PRIMARY KEY,
  slug           CITEXT       NOT NULL,
  title          VARCHAR(128) NOT NULL,
  owner_id       BIGINT
    CONSTRAINT forum_user_id_fk
    REFERENCES "user",
  owner_nickname CITEXT       NOT NULL,
  posts          BIGINT  DEFAULT 0,
  threads        INTEGER DEFAULT 0
);

-- CREATE INDEX IF NOT EXISTS forum_owner_id_idx
--   ON forum (owner_id);

CREATE UNIQUE INDEX IF NOT EXISTS forum_slug_uindex
  ON forum (slug);

CREATE TABLE IF NOT EXISTS users_forum
(
  id         BIGSERIAL NOT NULL CONSTRAINT users_forum_pkey
  PRIMARY KEY,
  forum_slug CITEXT NOT NULL,
  nickname   CITEXT NOT NULL,
  fullname   TEXT   NOT NULL,
  email      CITEXT NOT NULL,
  about      CITEXT NOT NULL
);

CREATE UNIQUE INDEX ON users_forum (forum_slug, nickname);

-- auto-generated definition
CREATE TABLE IF NOT EXISTS thread
(
  id             SERIAL                                         NOT NULL
    CONSTRAINT thread_pkey
    PRIMARY KEY,
  slug           CITEXT DEFAULT NULL :: CITEXT,
  title          VARCHAR(128) DEFAULT NULL :: CHARACTER VARYING NOT NULL,
  message        TEXT                                           NOT NULL,
  forum_id       BIGINT                                         NOT NULL
    CONSTRAINT thread_forum_id_fk
    REFERENCES forum,
  forum_slug     CITEXT                                         NOT NULL,
  owner_id       INTEGER                                        NOT NULL
    CONSTRAINT thread_user_id_fk
    REFERENCES "user",
  owner_nickname CITEXT                                         NOT NULL,
  created        TIMESTAMP WITH TIME ZONE DEFAULT now()         NOT NULL,
  votes          INTEGER DEFAULT 0                              NOT NULL
);

-- CREATE INDEX IF NOT EXISTS thread_forum_slug
--   ON thread (forum_slug);


CREATE UNIQUE INDEX IF NOT EXISTS thread_slug_uindex
  ON thread (slug);

CREATE INDEX IF NOT EXISTS thread_forum_id_idx
  ON thread (forum_id);

-- CREATE INDEX IF NOT EXISTS thread_owner_id_idx
--   ON thread (owner_id);

CREATE OR REPLACE FUNCTION update_count_threads()
  RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
  UPDATE forum
  SET threads = threads + 1
  WHERE forum.id = new.forum_id;
  RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS count_threads_tgr
ON thread;

CREATE TRIGGER count_threads_tgr
  BEFORE INSERT
  ON thread
  FOR EACH ROW
EXECUTE PROCEDURE update_count_threads();

-- auto-generated definition
CREATE TABLE IF NOT EXISTS post
(
  id             BIGSERIAL                                             NOT NULL
    CONSTRAINT post_pkey
    PRIMARY KEY,
  message        TEXT                                                  NOT NULL,
  thread_id      INTEGER                                               NOT NULL
    CONSTRAINT post_thread_id_fk
    REFERENCES thread,
  parent         BIGINT DEFAULT 0                                      NOT NULL,
  owner_id       BIGINT                                                NOT NULL
    CONSTRAINT post_user_id_fk
    REFERENCES "user",
  owner_nickname CITEXT                                                NOT NULL,
  forum_slug     CITEXT                                                NOT NULL,
  created        TIMESTAMP WITHOUT TIME ZONE DEFAULT current_timestamp NOT NULL,
  isedited       BOOLEAN DEFAULT FALSE                                 NOT NULL,
  path           BIGINT []
);

CREATE INDEX ON post (parent, thread_id);

CREATE INDEX IF NOT EXISTS post_forum_slug
  ON post (forum_slug);

CREATE INDEX IF NOT EXISTS post_thread_id_idx
  ON post (thread_id);

CREATE INDEX IF NOT EXISTS post_parent_id_idx
  ON post (parent);


CREATE INDEX IF NOT EXISTS post_owner_id_idx
  ON post (owner_id);

CREATE INDEX IF NOT EXISTS post_parent_path_idx
  ON post USING GIN (path);

CREATE INDEX IF NOT EXISTS post_parent_id_idx
  ON post (parent, path);

CREATE OR REPLACE FUNCTION update_section_parent_path()
  RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
  p BIGINT [];
BEGIN
  IF NEW.parent = 0
  THEN
    NEW.path = ARRAY [new.id];
  ELSEIF TG_OP = 'INSERT'
    THEN
      SELECT pst.path || new.id
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

DROP TRIGGER IF EXISTS parent_path_tgr
ON post;

CREATE TRIGGER parent_path_tgr
  BEFORE INSERT
  ON post
  FOR EACH ROW
EXECUTE PROCEDURE update_section_parent_path();

CREATE OR REPLACE FUNCTION update_users_forum_on_post()
  RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
  LOCK TABLE users_forum;
  INSERT INTO users_forum (forum_slug, nickname, fullname, email, about)
    (
      SELECT
        NEW.forum_slug,
        NEW.owner_nickname,
        u.fullname,
        u.email,
        u.about
      FROM "user" u
        WHERE u.id = new.owner_id FOR NO KEY UPDATE)
  ON CONFLICT DO NOTHING;
  RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION update_users_forum_on_thread()
  RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
  INSERT INTO users_forum (forum_slug, nickname, fullname, email, about)
    (
      SELECT
        NEW.forum_slug,
        NEW.owner_nickname,
        u.fullname,
        u.email,
        u.about
      FROM "user" u
        WHERE u.id = new.owner_id FOR NO KEY UPDATE)
  ON CONFLICT DO NOTHING;
  RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS update_users_posts_tgr
ON post;

-- CREATE TRIGGER update_users_posts_tgr
--   AFTER INSERT
--   ON post
--   FOR EACH ROW
-- EXECUTE PROCEDURE update_users_forum_on_post();

DROP TRIGGER IF EXISTS update_users_thread_tgr
ON thread;

CREATE TRIGGER update_users_thread_tgr
  AFTER INSERT
  ON thread
  FOR EACH ROW
EXECUTE PROCEDURE update_users_forum_on_thread();

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

CREATE UNIQUE INDEX IF NOT EXISTS votes_user_id_thread_id_uindex
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

DROP TRIGGER IF EXISTS update_count_votes_trig
ON votes;

CREATE TRIGGER update_count_votes_trig
  AFTER INSERT OR UPDATE
  ON votes
  FOR EACH ROW
EXECUTE PROCEDURE update_count_votes();


CREATE INDEX ON post (parent, thread_id, id);
CREATE INDEX ON post (thread_id, path);
CREATE INDEX on post (id ASC , thread_id ASC);
CREATE INDEX ON post (id ASC, forum_slug ASC);

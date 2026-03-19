-- read-only role

DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolename = 'readonly') THEN
    CREATE ROLE readonly;
  END IF;
END
$$;

GRANT CONNECT ON DATABASE gogenexample TO readonly;

-- read-write role

DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolename = 'readwrite') THEN
    CREATE ROLE readwrite;
  END IF;
END
$$;

GRANT CONNECT ON DATABASE gogenexample TO readwrite;

-- read-only user

DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolename = 'dbro_user') THEN
    CREATE USER dbro_user WITH PASSWORD 'dbro_pass';
  ELSE
    ALTER USER dbro_user WITH PASSWORD 'dbro_pass';
  END IF;
END
$$;


-- read-write user

DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolename = 'dbrw_user') THEN
    CREATE USER dbrw_user WITH PASSWORD 'dbrw_pass';
  ELSE
    ALTER USER dbrw_user WITH PASSWORD 'dbrw_pass';
  END IF;
END
$$;

-- grant role memberships to users
GRANT readonly TO dbro_user;
GRANT readwrite TO dbrw_user;

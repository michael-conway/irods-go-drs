CREATE DATABASE "ICAT";
CREATE USER irods WITH PASSWORD 'testpassword';
GRANT ALL PRIVILEGES ON DATABASE "ICAT" to irods;
-- Make irods the owner of the public schema
ALTER SCHEMA public OWNER TO irods;

-- Give irods full privileges on the schema
GRANT ALL ON SCHEMA public TO irods;

-- Make irods the owner of the database itself
ALTER DATABASE "ICAT" OWNER TO irods;

-- Optional: ensure irods can connect and create objects
GRANT ALL PRIVILEGES ON DATABASE "ICAT" TO irods;

CREATE DATABASE "KEYCLOAK";

-- Make irods the owner of the database itself
ALTER DATABASE "KEYCLOAK" OWNER TO irods;

-- Optional: ensure irods can connect and create objects
GRANT ALL PRIVILEGES ON DATABASE "KEYCLOAK" TO irods;

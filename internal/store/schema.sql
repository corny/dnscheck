-- Database schema, applied on every start. Every statement must be
-- idempotent so it can run on new and existing databases alike.
-- Taken over from the migrations of the former Rails application.

CREATE TABLE IF NOT EXISTS nameservers (
  id                bigserial PRIMARY KEY,
  ip_address        inet NOT NULL,
  name              varchar(255),
  as_number         integer,
  as_org            varchar(255),
  country_code      varchar(2),
  city              varchar(255),
  status            boolean NOT NULL,
  status_changed_at timestamp,
  error             varchar(255),
  version           varchar(255),
  reliability       double precision,
  dnssec            boolean,
  checked_at        timestamp,
  created_at        timestamp NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS index_nameservers_on_ip_address ON nameservers (ip_address);
CREATE INDEX IF NOT EXISTS index_nameservers_on_as_number ON nameservers (as_number);
CREATE INDEX IF NOT EXISTS index_nameservers_on_status ON nameservers (status);
CREATE INDEX IF NOT EXISTS index_nameservers_on_checked_at ON nameservers (checked_at);
CREATE INDEX IF NOT EXISTS index_nameservers_on_country_code_and_status_and_checked_at ON nameservers (country_code, status, checked_at);

CREATE TABLE IF NOT EXISTS nameserver_checks (
  nameserver_id bigint NOT NULL REFERENCES nameservers (id) ON DELETE CASCADE,
  result        boolean NOT NULL,
  error         varchar(255),
  created_at    timestamp NOT NULL
);

CREATE INDEX IF NOT EXISTS index_nameserver_checks_on_nameserver_id ON nameserver_checks (nameserver_id);

-- Every change of checked_at records a check result in nameserver_checks
-- and recomputes the reliability. A change of status sets status_changed_at.
CREATE OR REPLACE FUNCTION insert_nameserver_check() RETURNS trigger AS $$
  BEGIN
    IF NEW.checked_at != OLD.checked_at THEN
        INSERT INTO nameserver_checks (nameserver_id, result, error, created_at) VALUES (NEW.id, NEW.status, NEW.error, NEW.checked_at);
        SELECT COUNT(CASE WHEN result THEN 1 END)::float/COUNT(*) INTO NEW.reliability FROM nameserver_checks WHERE nameserver_id=NEW.id;
    END IF;

    IF OLD IS NULL THEN
        INSERT INTO nameserver_checks (nameserver_id, result, error, created_at) VALUES (NEW.id, NEW.status, NEW.error, NEW.checked_at);
    END IF;

    IF NEW.status != OLD.status THEN
        NEW.status_changed_at = NEW.checked_at;
    END IF;

    RETURN NEW;
  END
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER insert_checks_on_update
  BEFORE UPDATE ON nameservers
  FOR EACH ROW
  EXECUTE FUNCTION insert_nameserver_check();

CREATE OR REPLACE TRIGGER insert_checks_on_insert
  AFTER INSERT ON nameservers
  FOR EACH ROW
  EXECUTE FUNCTION insert_nameserver_check();

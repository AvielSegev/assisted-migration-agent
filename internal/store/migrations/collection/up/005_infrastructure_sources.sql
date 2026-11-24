-- Resource names and VMware object IDs are only unique within a vCenter.
-- Keep the source when single-file RVTools databases are appended together.
ALTER TABLE IF EXISTS vhost ADD COLUMN IF NOT EXISTS source_vcenter_id VARCHAR;
ALTER TABLE IF EXISTS vdatastore ADD COLUMN IF NOT EXISTS source_vcenter_id VARCHAR;
ALTER TABLE IF EXISTS vnic ADD COLUMN IF NOT EXISTS source_vcenter_id VARCHAR;
ALTER TABLE IF EXISTS vhba ADD COLUMN IF NOT EXISTS source_vcenter_id VARCHAR;
ALTER TABLE IF EXISTS vhostscsi ADD COLUMN IF NOT EXISTS source_vcenter_id VARCHAR;

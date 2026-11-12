-- The version id each stream last reported running on check-in, as the node sent it.
-- NULL = none: never reported, or the node reported that it runs no known version.
ALTER TABLE nodes ADD COLUMN reported_config_version_id TEXT;
ALTER TABLE nodes ADD COLUMN reported_binary_version_id TEXT;

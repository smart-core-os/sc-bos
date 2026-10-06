-- Nodes record whether they can install software (binary) updates, as last reported on check-in.
-- NULL = unknown: never reported, or the node's build doesn't report it.
ALTER TABLE nodes ADD COLUMN binary_updates BOOLEAN;

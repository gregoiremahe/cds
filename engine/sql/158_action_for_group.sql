-- +migrate Up

-- remove unused table and column
DROP TABLE IF EXISTS template_action;
ALTER TABLE action_parameter DROP COLUMN IF EXISTS worker_model_name;
ALTER TABLE "action" DROP COLUMN public;

-- replace existing foreign keys with cascade ones
ALTER TABLE action_parameter DROP CONSTRAINT "fk_action_parameter_action";
ALTER TABLE action_requirement DROP CONSTRAINT "fk_action_requirement_action";
ALTER TABLE action_edge DROP CONSTRAINT "fk_action_edge_parent_action";
ALTER TABLE action_edge_parameter DROP CONSTRAINT "fk_action_edge_parameter_action_edge";
ALTER TABLE pipeline_action DROP CONSTRAINT "fk_pipeline_action_action";
SELECT create_foreign_key_idx_cascade('FK_ACTION_PARAMETER_ACTION', 'action_parameter', 'action', 'action_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_ACTION_REQUIREMENT_ACTION', 'action_requirement', 'action', 'action_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_ACTION_EDGE_PARENT_ACTION', 'action_edge', 'action', 'parent_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_ACTION_EDGE_PARAMETER_ACTION_EDGE', 'action_edge_parameter', 'action_edge', 'action_edge_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_PIPELINE_ACTION_ACTION', 'pipeline_action', 'action', 'action_id', 'id');

-- change type for column name and type and add indexes (usefull to check if action exists)
ALTER TABLE "action" ALTER COLUMN "name" TYPE VARCHAR(100) USING "name"::VARCHAR(100);
CREATE INDEX idx_action_name ON "action" ("name");
ALTER TABLE "action" ALTER COLUMN "type" TYPE VARCHAR(100) USING "type"::VARCHAR(100);
CREATE INDEX idx_action_type ON "action" ("type");

-- add column group_id on action and set to 1 (shared.infra) for all not joined action
ALTER TABLE "action" ADD COLUMN group_id BIGINT;
SELECT create_foreign_key_idx_cascade('FK_ACTION_GROUP', 'action', 'group', 'group_id', 'id');
UPDATE "action" SET group_id = 1 WHERE "type" <> 'Joined';

-- +migrate Down

-- restore public column
ALTER TABLE "action" ADD COLUMN public BOOLEAN NOT NULL DEFAULT true;

-- restore foreign keys
ALTER TABLE action_parameter DROP CONSTRAINT "fk_action_parameter_action";
ALTER TABLE action_requirement DROP CONSTRAINT "fk_action_requirement_action";
ALTER TABLE action_edge DROP CONSTRAINT "fk_action_edge_parent_action";
ALTER TABLE action_edge_parameter DROP CONSTRAINT "fk_action_edge_parameter_action_edge";
ALTER TABLE pipeline_action DROP CONSTRAINT "fk_pipeline_action_action";
select create_foreign_key('FK_ACTION_PARAMETER_ACTION', 'action_parameter', 'action', 'action_id', 'id');
select create_foreign_key('FK_ACTION_REQUIREMENT_ACTION', 'action_requirement', 'action', 'action_id', 'id');
select create_foreign_key('FK_ACTION_EDGE_PARENT_ACTION', 'action_edge', 'action', 'parent_id', 'id');
select create_foreign_key('FK_ACTION_EDGE_PARAMETER_ACTION_EDGE', 'action_edge_parameter', 'action_edge', 'action_edge_id', 'id');
select create_foreign_key('FK_PIPELINE_ACTION_ACTION', 'pipeline_action', 'action', 'action_id', 'id');

-- restore type for column name and type and remove indexes
ALTER TABLE "action" ALTER COLUMN "name" TYPE TEXT USING "name"::TEXT;
DROP INDEX IF EXISTS idx_action_name;
ALTER TABLE "action" ALTER COLUMN "type" TYPE TEXT USING "type"::TEXT;
DROP INDEX IF EXISTS idx_action_type;

-- remove group_id column
ALTER TABLE "action" DROP COLUMN group_id;

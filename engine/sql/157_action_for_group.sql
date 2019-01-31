-- +migrate Up

DROP TABLE template_action;
ALTER TABLE action_parameter DROP COLUMN worker_model_name;

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

-- +migrate Down

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

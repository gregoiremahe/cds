package action

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// Usage for action.
type Usage struct {
	Pipelines []UsagePipeline `json:"pipelines"`
	Actions   []UsageAction   `json:"audits"`
}

// UsagePipeline represent a pipeline using an action.
type UsagePipeline struct {
	ProjectID    string `json:"project_id"`
	ProjectKey   string `json:"project_key"`
	ProjectName  string `json:"project_name"`
	PipelineID   int64  `json:"pipeline_id"`
	PipelineName string `json:"pipeline_name"`
	StageID      int64  `json:"stage_id"`
	StageName    string `json:"stage_Name"`
	JobID        int64  `json:"job_id"`
	JobName      string `json:"job_name"`
	ActionID     int64  `json:"action_id"`
	ActionName   string `json:"action_name"`
	Warning      bool   `json:"warning"`
}

// GetPipelineUsages returns the list of pipelines using an action
func GetPipelineUsages(db gorp.SqlExecutor, sharedInfraGroupID, actionID int64) ([]UsagePipeline, error) {
	rows, err := db.Query(`
    SELECT
      project.id as projectId, project.projectKey as projectKey, project.name as projectName,
      pipeline.id as pipelineId, pipeline.name as pipelineName,
      pipeline_stage.id as stageId, pipeline_stage.name as stageName,
      parent.id as jobId, parent.name as jobName,
      action.id as actionId, action.name as actionName,
      CAST((CASE WHEN project_group.role IS NOT NULL OR action.group_id = $1 THEN 0 ELSE 1 END) AS BIT) as warning
		FROM action
    INNER JOIN action_edge ON action_edge.child_id = action.id
    LEFT JOIN action as parent ON parent.id = action_edge.parent_id
		INNER JOIN pipeline_action ON pipeline_action.action_id = parent.id
		LEFT JOIN pipeline_stage ON pipeline_stage.id = pipeline_action.pipeline_stage_id
		LEFT JOIN pipeline ON pipeline.id = pipeline_stage.pipeline_id
    LEFT JOIN project ON pipeline.project_id = project.id
    LEFT JOIN project_group ON project_group.project_id = project.id AND project_group.group_id = action.group_id
		WHERE action.id = $2
		ORDER BY projectkey, pipelineName, actionName;
	`, sharedInfraGroupID, actionID)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot load pipeline usages for action with id %d", actionID)
	}
	defer rows.Close()

	us := []UsagePipeline{}
	for rows.Next() {
		var u UsagePipeline
		if err := rows.Scan(
			&u.ProjectID, &u.ProjectKey, &u.ProjectName,
			&u.PipelineID, &u.PipelineName,
			&u.StageID, &u.StageName,
			&u.ActionID, &u.ActionName,
			&u.JobID, &u.JobName,
			&u.Warning,
		); err != nil {
			return nil, sdk.WrapError(err, "cannot scan sql rows")
		}
		us = append(us, u)
	}

	return us, nil
}

// UsageAction represent a action using an action.
type UsageAction struct {
	ParentActionID   int64  `json:"parent_action_id"`
	ParentActionName string `json:"parent_action_name"`
	ActionID         int64  `json:"action_id"`
	ActionName       string `json:"action_name"`
	Warning          bool   `json:"warning"`
}

// GetActionUsages returns the list of actions using an action
func GetActionUsages(db gorp.SqlExecutor, sharedInfraGroupID, actionID int64) ([]UsageAction, error) {
	rows, err := db.Query(`
    SELECT
      parent.id as parentActionId, parent.name as parentActionName,
      action.id as actionId, action.name as actionName,
      CAST((CASE WHEN action.group_id = parent.group_id OR action.group_id = $1 THEN 0 ELSE 1 END) AS BIT) as warning
		FROM action
		INNER JOIN action_edge ON action_edge.child_id = action.id
		LEFT JOIN action as parent ON parent.id = action_edge.parent_id
		WHERE action.id = $2 AND parent.group_id IS NOT NULL
		ORDER BY parentActionName, actionName;
	`, sharedInfraGroupID, actionID)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot load pipeline usages for action with id %d", actionID)
	}
	defer rows.Close()

	us := []UsageAction{}
	for rows.Next() {
		var u UsageAction
		if err := rows.Scan(
			&u.ParentActionID, &u.ParentActionName,
			&u.ActionID, &u.ActionName,
			&u.Warning,
		); err != nil {
			return nil, sdk.WrapError(err, "cannot scan sql rows")
		}
		us = append(us, u)
	}

	return us, nil
}

// PipelineUsingAction represent a pipeline using an action
type PipelineUsingAction struct {
	ActionID         int    `json:"action_id"`
	ActionType       string `json:"type"`
	ActionName       string `json:"action_name"`
	PipName          string `json:"pipeline_name"`
	AppName          string `json:"application_name"`
	EnvID            int64  `json:"environment_id"`
	ProjName         string `json:"project_name"`
	ProjKey          string `json:"key"`
	StageID          int64  `json:"stage_id"`
	WorkflowName     string `json:"workflow_name"`
	WorkflowNodeName string `json:"workflow_node_name"`
	WorkflowNodeID   int64  `json:"workflow_node_id"`
}

// GetPipelineUsingAction returns the list of pipelines using an action
func GetPipelineUsingAction(db gorp.SqlExecutor, name string) ([]PipelineUsingAction, error) {
	query := `
		SELECT
			action.type, action.name as actionName, action.id as actionId,
			pipeline_stage.id as stageId,
			pipeline.name as pipName, project.name, project.projectkey,
			workflow.name as wName, workflow_node.id as nodeId,  workflow_node.name as nodeName
		FROM action_edge
		LEFT JOIN action on action.id = parent_id
		LEFT OUTER JOIN pipeline_action ON pipeline_action.action_id = action.id
		LEFT OUTER JOIN pipeline_stage ON pipeline_stage.id = pipeline_action.pipeline_stage_id
		LEFT OUTER JOIN pipeline ON pipeline.id = pipeline_stage.pipeline_id
		LEFT OUTER JOIN project ON pipeline.project_id = project.id
		LEFT OUTER JOIN workflow_node ON workflow_node.pipeline_id = pipeline.id
		LEFT OUTER JOIN workflow ON workflow_node.workflow_id = workflow.id
		LEFT JOIN action as actionChild ON  actionChild.id = child_id
		WHERE actionChild.name = $1 and actionChild.public = true AND pipeline.name IS NOT NULL
		ORDER BY projectkey, pipName, actionName;
	`
	rows, errq := db.Query(query, name)
	if errq != nil {
		return nil, sdk.WrapError(errq, "getPipelineUsingAction> Cannot load pipelines using action %s", name)
	}
	defer rows.Close()

	response := []PipelineUsingAction{}
	for rows.Next() {
		var a PipelineUsingAction
		var pipName, projName, projKey, wName, wnodeName sql.NullString
		var stageID, nodeID sql.NullInt64
		if err := rows.Scan(&a.ActionType, &a.ActionName, &a.ActionID, &stageID,
			&pipName, &projName, &projKey,
			&wName, &nodeID, &wnodeName,
		); err != nil {
			return nil, sdk.WrapError(err, "Cannot read sql response")
		}
		if stageID.Valid {
			a.StageID = stageID.Int64
		}
		if pipName.Valid {
			a.PipName = pipName.String
		}
		if projName.Valid {
			a.ProjName = projName.String
		}
		if projKey.Valid {
			a.ProjKey = projKey.String
		}
		if wName.Valid {
			a.WorkflowName = wName.String
		}
		if wnodeName.Valid {
			a.WorkflowNodeName = wnodeName.String
		}
		if nodeID.Valid {
			a.WorkflowNodeID = nodeID.Int64
		}

		response = append(response, a)
	}

	return response, nil
}

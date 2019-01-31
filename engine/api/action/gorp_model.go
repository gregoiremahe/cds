package action

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

type actionParameter struct {
	sdk.Parameter
	ActionID int64 `db:"action_id"`
}

func actionParametersToParameters(aps []actionParameter) []sdk.Parameter {
	ps := make([]sdk.Parameter, len(aps))
	for i := range aps {
		ps[i] = aps[i].Parameter
	}
	return ps
}

type actionEdge struct {
	ID             int64  `db:"id"`
	ParentID       int64  `db:"parent_id"`
	ChildID        int64  `db:"child_id"`
	ExecOrder      int64  `db:"exec_order"`
	Enabled        bool   `db:"enabled"`
	Optional       bool   `db:"optional"`
	AlwaysExecuted bool   `db:"always_executed"`
	StepName       string `db:"step_name"`
	// aggregates
	Parameters []actionEdgeParameter `db:"-"`
	Child      *sdk.Action           `db:"-"`
}

func actionEdgesToIDs(aes []*actionEdge) []int64 {
	var ids []int64
	for i := range aes {
		ids = append(ids, aes[i].ID)
	}
	return ids
}

func actionEdgesToChildIDs(aes []*actionEdge) []int64 {
	var ids []int64
	for i := range aes {
		ids = append(ids, aes[i].ChildID)
	}
	return ids
}

type actionEdgeParameter struct {
	sdk.Parameter
	ActionEdgeID int64 `db:"action_edge_id"`
}

func init() {
	gorpmapping.Register(
		gorpmapping.New(sdk.Action{}, "action", true, "id"),
		gorpmapping.New(sdk.ActionAudit{}, "action_audit", false),
		gorpmapping.New(actionParameter{}, "action_parameter", true, "id"),
		gorpmapping.New(sdk.Requirement{}, "action_requirement", true, "id"),
		gorpmapping.New(actionEdge{}, "action_edge", true, "id"),
		gorpmapping.New(actionEdgeParameter{}, "action_edge_parameter", true, "id"),
	)
}

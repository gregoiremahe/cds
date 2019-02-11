package action

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

func getAll(db gorp.SqlExecutor, q gorpmapping.Query, ags ...actionAggregator) ([]sdk.Action, error) {
	pas := []*sdk.Action{}

	if err := gorpmapping.GetAll(db, q, &pas); err != nil {
		return nil, sdk.WrapError(err, "cannot get actions")
	}
	if len(ags) > 0 {
		for i := range ags {
			if err := ags[i](db, pas...); err != nil {
				return nil, err
			}
		}
	}
	if len(pas) == 0 {
		return nil, sdk.WithStack(sdk.ErrNoAction)
	}

	as := make([]sdk.Action, len(pas))
	for i := range pas {
		as[i] = *pas[i]
	}

	return as, nil
}

func get(db gorp.SqlExecutor, q gorpmapping.Query, ags ...actionAggregator) (*sdk.Action, error) {
	var a sdk.Action

	found, err := gorpmapping.Get(db, q, &a)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get action")
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNoAction)
	}

	for i := range ags {
		if err := ags[i](db, &a); err != nil {
			return nil, err
		}
	}

	return &a, nil
}

// insert action in database.
func insert(db gorp.SqlExecutor, a *sdk.Action) error {
	return sdk.WithStack(db.Insert(a))
}

// DeleteRequirementsByActionID deletes all requirements related to given action.
func DeleteRequirementsByActionID(db gorp.SqlExecutor, actionID int64) error {
	_, err := db.Exec("DELETE FROM action_requirement WHERE action_id = $1", actionID)
	return sdk.WithStack(err)
}

func getRequirementsByActionIDs(db gorp.SqlExecutor, actionIDs []int64) ([]sdk.Requirement, error) {
	var rs []sdk.Requirement

	query := gorpmapping.NewQuery(
		"SELECT * FROM action_requirement WHERE action_id = ANY(string_to_array($1, ',')::int[]) ORDER BY name",
	).Args(gorpmapping.IDsToQueryString(actionIDs))
	if err := gorpmapping.GetAll(db, query, &rs); err != nil {
		return nil, sdk.WrapError(err, "cannot get requirements for action ids %v", actionIDs)
	}

	return rs, nil
}

// GetRequirementsDistinctBinary retrieves all binary requirements in database.
// Used by worker to automatically declare most capabilities, this func returns denormalized values.
func GetRequirementsDistinctBinary(db gorp.SqlExecutor) (sdk.RequirementList, error) {
	rows, err := db.Query("SELECT distinct value FROM action_requirement where type = 'binary'")
	if err != nil {
		return nil, sdk.WithStack(err)
	}
	defer rows.Close()

	var rs []sdk.Requirement
	var value string
	for rows.Next() {
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		rs = append(rs, sdk.Requirement{
			Name:  value,
			Type:  sdk.BinaryRequirement,
			Value: value,
		})
	}

	return rs, nil
}

// InsertRequirement in database.
func InsertRequirement(db gorp.SqlExecutor, r *sdk.Requirement) error {
	if r.Name == "" || r.Type == "" || r.Value == "" {
		return sdk.WithStack(sdk.ErrInvalidJobRequirement)
	}
	return sdk.WithStack(db.Insert(r))
}

// UpdateRequirementsValue updates all action_requirement.value given a value and a type then returns action IDs.
func UpdateRequirementsValue(db gorp.SqlExecutor, oldValue, newValue, reqType string) ([]int64, error) {
	rows, err := db.Query("UPDATE action_requirement SET value = $1 WHERE value = $2 AND type = $3 RETURNING action_id", newValue, oldValue, reqType)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot update action requirements (newValue=%s, oldValue=%s, reqType=%v)", newValue, oldValue, reqType)
	}
	defer rows.Close()

	var actionID int64
	var actionIDs = []int64{}
	for rows.Next() {
		if err := rows.Scan(&actionID); err != nil {
			return nil, sdk.WrapError(err, "unable to scan action id")
		}
		actionIDs = append(actionIDs, actionID)
	}

	return actionIDs, nil
}

func getParametersByActionIDs(db gorp.SqlExecutor, actionIDs []int64) ([]actionParameter, error) {
	var aps []actionParameter

	query := gorpmapping.NewQuery(
		"SELECT * FROM action_parameter WHERE action_id = ANY(string_to_array($1, ',')::int[]) ORDER BY name",
	).Args(gorpmapping.IDsToQueryString(actionIDs))
	if err := gorpmapping.GetAll(db, query, &aps); err != nil {
		return nil, sdk.WrapError(err, "cannot get parameters for action ids %v", actionIDs)
	}

	return aps, nil
}

func insertParameter(db gorp.SqlExecutor, p *actionParameter) error {
	if string(p.Type) == string(sdk.SecretVariable) {
		return sdk.WithStack(sdk.ErrNoDirectSecretUse)
	}
	return sdk.WrapError(db.Insert(p), "unable to insert parameter for action %d", p.ActionID)
}

func deleteParametersByActionID(db gorp.SqlExecutor, actionID int64) error {
	_, err := db.Exec("DELETE FROM action_parameter WHERE action_id = $1", actionID)
	return sdk.WithStack(err)
}

func getEdges(db gorp.SqlExecutor, q gorpmapping.Query, ags ...edgeAggregator) ([]actionEdge, error) {
	paes := []*actionEdge{}

	if err := gorpmapping.GetAll(db, q, &paes); err != nil {
		return nil, sdk.WrapError(err, "cannot get action edges")
	}
	if len(paes) > 0 {
		for i := range ags {
			if err := ags[i](db, paes...); err != nil {
				return nil, err
			}
		}
	}

	aes := make([]actionEdge, len(paes))
	for i := range paes {
		aes[i] = *paes[i]
	}

	return aes, nil
}

func insertEdge(db gorp.SqlExecutor, ae *actionEdge) error {
	return sdk.WrapError(gorpmapping.Insert(db, ae), "unable to insert action edge for parent %d and child %d", ae.ParentID, ae.ChildID)
}

func getEdgeParametersByEdgeIDs(db gorp.SqlExecutor, edgesIDs []int64) ([]actionEdgeParameter, error) {
	aeps := []actionEdgeParameter{}

	query := gorpmapping.NewQuery(
		"SELECT * FROM action_edge_parameter WHERE action_edge_id = ANY(string_to_array($1, ',')::int[]) ORDER BY name",
	).Args(gorpmapping.IDsToQueryString(edgesIDs))
	if err := gorpmapping.GetAll(db, query, &aeps); err != nil {
		return nil, sdk.WrapError(err, "cannot get action edge parameters for edge ids %d", edgesIDs)
	}

	return aeps, nil
}

func insertEdgeParameter(db gorp.SqlExecutor, aep *actionEdgeParameter) error {
	return sdk.WrapError(gorpmapping.Insert(db, aep), "unable to insert action edge parameter for edge %d", aep.ActionEdgeID)
}

// deleteEdgesByParentID delete all action edge in database for a given parentID
func deleteEdgesByParentID(db gorp.SqlExecutor, parentID int64) error {
	_, err := db.Exec("DELETE FROM action_edge WHERE parent_id = $1", parentID)
	return sdk.WithStack(err)
}

package action

import (
	"encoding/json"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// CheckChildrenForGroupIDs returns an error if given children not found.
func CheckChildrenForGroupIDs(db gorp.SqlExecutor, a *sdk.Action, groupIDs []int64) error {
	if len(a.Actions) == 0 {
		return nil
	}

	childrenIDs := a.ToUniqueChildrenIDs()
	children, err := loadAllTypeBuiltInOrDefaultByIDsAndGroupIDs(db, childrenIDs, groupIDs)
	if err != nil {
		return err
	}
	if len(children) != len(childrenIDs) {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "some given child can not be found")
	}

	return nil
}

// CheckChildrenForGroupIDsWithLoop return an error if given children not found or tree loop detected.
func CheckChildrenForGroupIDsWithLoop(db gorp.SqlExecutor, a *sdk.Action, groupIDs []int64) error {
	if len(a.Actions) == 0 {
		return nil
	}

	// if builtin, it has no children so it's ok
	if a.Type == sdk.BuiltinAction {
		return nil
	}

	childrenIDs := a.ToUniqueChildrenIDs()
	children, err := loadAllTypeBuiltInOrDefaultByIDsAndGroupIDs(db, childrenIDs, groupIDs)
	if err != nil {
		return err
	}
	if len(children) != len(childrenIDs) {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "some given child can not be found")
	}

	for i := range children {
		if err := CheckChildrenForGroupIDsWithLoop(db, &children[i], groupIDs); err != nil {
			return err
		}
	}

	return nil
}

// Insert insert given action and its components in database.
func Insert(db gorp.SqlExecutor, a *sdk.Action) error {
	// insert the action and its components
	if err := insert(db, a); err != nil {
		return err
	}

	for i := range a.Actions {
		if err := insertActionChild(db, a.Actions[i], a.ID, i+1); err != nil {
			return err
		}
	}

	// Requirements of children are requirement of parent
	for _, c := range a.Actions {
		if len(c.Requirements) == 0 {
			log.Debug("Try load children action requirement for id:%d", c.ID)
			var err error
			c.Requirements, err = getRequirementsByActionIDs(db, []int64{c.ID})
			if err != nil {
				return err
			}
		}
		// Now for each requirement of child, check if it exists in parent
		for _, cr := range c.Requirements {
			found := false
			for _, pr := range a.Requirements {
				if pr.Type == cr.Type && pr.Value == cr.Value {
					found = true
					break
				}
			}
			if !found {
				a.Requirements = append(a.Requirements, cr)
			}
		}
	}

	if err := isRequirementsValid(a.Requirements); err != nil {
		return err
	}

	for i := range a.Requirements {
		r := a.Requirements[i]
		r.ActionID = a.ID
		if err := InsertRequirement(db, &r); err != nil {
			return err
		}
	}

	for i := range a.Parameters {
		if err := insertParameter(db, &actionParameter{
			ActionID:    a.ID,
			Name:        a.Parameters[i].Name,
			Type:        a.Parameters[i].Type,
			Value:       a.Parameters[i].Value,
			Description: a.Parameters[i].Description,
			Advanced:    a.Parameters[i].Advanced,
		}); err != nil {
			return sdk.WrapError(err, "cannot insert action parameter %s", a.Parameters[i].Name)
		}
	}

	return nil
}

// UpdateActionDB  Update an action
func UpdateActionDB(db gorp.SqlExecutor, a *sdk.Action, userID int64) error {
	if err := insertAudit(db, a.ID, userID, "action update"); err != nil {
		return err
	}

	if err := deleteEdgesByParentID(db, a.ID); err != nil {
		return err
	}
	for i := range a.Actions {
		if err := insertActionChild(db, a.Actions[i], a.ID, i+1); err != nil {
			return err
		}
	}

	if err := deleteParametersByActionID(db, a.ID); err != nil {
		return err
	}
	for i := range a.Parameters {
		if err := insertParameter(db, &actionParameter{
			ActionID:    a.ID,
			Name:        a.Parameters[i].Name,
			Type:        a.Parameters[i].Type,
			Value:       a.Parameters[i].Value,
			Description: a.Parameters[i].Description,
			Advanced:    a.Parameters[i].Advanced,
		}); err != nil {
			return sdk.WrapError(err, "insertActionParameter for %s failed", a.Parameters[i].Name)
		}
	}

	if err := DeleteRequirementsByActionID(db, a.ID); err != nil {
		return err
	}

	//TODO we don't need to compute all job requirements here, but only when running the job
	// Requirements of children are requirement of parent
	computeRequirements(a)

	// Checks if multiple requirements have the same name
	if err := isRequirementsValid(a.Requirements); err != nil {
		return err
	}

	for i := range a.Requirements {
		r := a.Requirements[i]
		r.ActionID = a.ID
		if err := InsertRequirement(db, &r); err != nil {
			return err
		}
	}

	query := `UPDATE action SET name=$1, description=$2, type=$3, enabled=$4, deprecated=$5 WHERE id=$6`
	_, errdb := db.Exec(query, a.Name, a.Description, string(a.Type), a.Enabled, a.Deprecated, a.ID)
	return sdk.WithStack(errdb)
}

// DeleteAction remove action from database
func DeleteAction(db gorp.SqlExecutor, actionID, userID int64) error {
	if err := insertAudit(db, actionID, userID, "Action delete"); err != nil {
		return err
	}

	if _, err := db.Exec(`DELETE FROM action WHERE action.id = $1`, actionID); err != nil {
		return sdk.WithStack(err)
	}

	return nil
}

// Used checks if action is used in another action or in a pipeline
func Used(db gorp.SqlExecutor, actionID int64) (bool, error) {
	var count int

	if err := db.QueryRow(`SELECT COUNT(id) FROM pipeline_action WHERE action_id = $1`, actionID).Scan(&count); err != nil {
		return false, sdk.WithStack(err)
	}
	if count > 0 {
		return true, nil
	}

	if err := db.QueryRow(`SELECT COUNT(id) FROM action_edge WHERE child_id = $1`, actionID).Scan(&count); err != nil {
		return false, sdk.WithStack(err)
	}

	return count > 0, nil
}

func insertAudit(db gorp.SqlExecutor, actionID, userID int64, change string) error {
	a, errLoad := LoadByID(db, actionID)
	if errLoad != nil {
		return errLoad
	}

	query := `INSERT INTO action_audit (action_id, user_id, change, versionned, action_json)
			VALUES ($1, $2, $3, NOW(), $4)`

	b, errJSON := json.Marshal(a)
	if errJSON != nil {
		return errJSON
	}

	if _, err := db.Exec(query, actionID, userID, change, b); err != nil {
		return err
	}

	return nil
}

func isRequirementsValid(requirements sdk.RequirementList) error {
	nbModelReq, nbHostnameReq := 0, 0
	for i := range requirements {
		for j := range requirements {
			if requirements[i].Name == requirements[j].Name && requirements[i].Type == requirements[j].Type && i != j {
				return sdk.WrapError(sdk.ErrInvalidJobRequirement, "for requirement name %s and type %s", requirements[i].Name, requirements[i].Type)
			}
		}
		switch requirements[i].Type {
		case sdk.ModelRequirement:
			nbModelReq++
		case sdk.HostnameRequirement:
			nbHostnameReq++
		}
	}
	if nbModelReq > 1 {
		return sdk.ErrInvalidJobRequirementDuplicateModel
	}
	if nbHostnameReq > 1 {
		return sdk.ErrInvalidJobRequirementDuplicateHostname
	}
	return nil
}

func computeRequirements(a *sdk.Action) {
	if a.Enabled {
		// Requirements of children are requirement of parent
		for _, c := range a.Actions {
			if !c.Enabled { // If action is not enabled we don't need their requirements
				continue
			}
			// Now for each requirement of child, check if it exists in parent
			for _, cr := range c.Requirements {
				found := false
				for _, pr := range a.Requirements {
					if pr.Type == cr.Type && pr.Value == cr.Value {
						found = true
						break
					}
				}
				if !found {
					a.Requirements = append(a.Requirements, cr)
				}
			}
		}
	} else {
		a.Requirements = []sdk.Requirement{}
	}
}

package action

import (
	"fmt"
	"strings"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

func insertActionChild(db gorp.SqlExecutor, actionID int64, child sdk.Action, execOrder int) error {
	if child.ID == 0 {
		return sdk.WithStack(fmt.Errorf("child action has no id"))
	}

	// useful to not save a step_name if it's the same than the default name (for ascode)
	if strings.ToLower(child.Name) == strings.ToLower(child.StepName) {
		child.StepName = ""
	}

	ae := actionEdge{
		ParentID:       actionID,
		ChildID:        child.ID,
		ExecOrder:      int64(execOrder), // TODO exec order can be int 64
		StepName:       child.StepName,
		Optional:       child.Optional,
		AlwaysExecuted: child.AlwaysExecuted,
		Enabled:        child.Enabled,
	}
	if err := insertEdge(db, &ae); err != nil {
		return err
	}

	// insert all parameters
	for i := range child.Parameters {
		// default value for parameter type list should be the first item ("aa;bb;cc" -> "aa")
		if child.Parameters[i].Type == sdk.ListParameter && strings.Contains(child.Parameters[i].Value, ";") {
			child.Parameters[i].Value = strings.Split(child.Parameters[i].Value, ";")[0]
		}

		if err := insertEdgeParameter(db, &actionEdgeParameter{
			ActionEdgeID: ae.ID,
			Name:         child.Parameters[i].Name,
			Type:         child.Parameters[i].Type,
			Value:        child.Parameters[i].Value,
			Description:  child.Parameters[i].Description,
			Advanced:     child.Parameters[i].Advanced,
		}); err != nil {
			return err
		}
	}

	return nil
}

package action

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

type actionAggregator func(gorp.SqlExecutor, ...*sdk.Action) error

func aggregateActionRequirements(db gorp.SqlExecutor, as ...*sdk.Action) error {
	rs, err := getRequirementsByActionIDs(db, sdk.ActionsToIDs(as))
	if err != nil {
		return err
	}

	m := make(map[int64][]sdk.Requirement)
	for i := range rs {
		if _, ok := m[rs[i].ActionID]; !ok {
			m[rs[i].ActionID] = make([]sdk.Requirement, 0)
		}
		m[rs[i].ActionID] = append(m[rs[i].ActionID], rs[i])
	}
	for i := range as {
		if rs, ok := m[as[i].ID]; ok {
			as[i].Requirements = rs
		}
	}

	return nil
}

func aggregateActionParameters(db gorp.SqlExecutor, as ...*sdk.Action) error {
	ps, err := getParametersByActionIDs(db, sdk.ActionsToIDs(as))
	if err != nil {
		return err
	}

	m := make(map[int64][]actionParameter)
	for i := range ps {
		if _, ok := m[ps[i].ActionID]; !ok {
			m[ps[i].ActionID] = make([]actionParameter, 0)
		}
		m[ps[i].ActionID] = append(m[ps[i].ActionID], ps[i])
	}
	for i := range as {
		if ps, ok := m[as[i].ID]; ok {
			as[i].Parameters = actionParametersToParameters(ps)
		}
	}

	return nil
}

func aggregateActionChildren(db gorp.SqlExecutor, as ...*sdk.Action) error {
	// don't try to load children if action is builtin
	actionsNotBuiltIn := sdk.ActionsFilterNotTypes(as, sdk.BuiltinAction)
	if len(actionsNotBuiltIn) == 0 {
		return nil
	}

	// get edges for all actions, then init a map of edges for all actions
	edges, err := loadEdgesByParentIDs(db, sdk.ActionsToIDs(actionsNotBuiltIn))
	if err != nil {
		return err
	}
	mEdges := make(map[int64][]actionEdge)
	for i := range edges {
		if _, ok := mEdges[edges[i].ParentID]; !ok {
			mEdges[edges[i].ParentID] = make([]actionEdge, 0)
		}
		mEdges[edges[i].ParentID] = append(mEdges[edges[i].ParentID], edges[i])
	}

	// for all actions set children from its edges
	for i := range actionsNotBuiltIn {
		edges, ok := mEdges[actionsNotBuiltIn[i].ID]
		if !ok {
			continue
		}

		var children []sdk.Action
		for i := range edges {
			// init child from edge child then override with edge attributes and parameters
			child := *edges[i].Child
			child.StepName = edges[i].StepName
			child.Optional = edges[i].Optional
			child.AlwaysExecuted = edges[i].AlwaysExecuted
			child.Enabled = edges[i].Enabled

			// replace action parameter with value configured by user when he created the child action
			params := make([]sdk.Parameter, len(child.Parameters))
			for j := range child.Parameters {
				params[j] = child.Parameters[j]
				for k := range edges[i].Parameters {
					if edges[i].Parameters[k].Name == params[j].Name {
						params[j].Value = edges[i].Parameters[k].Value
						break
					}
				}
			}
			child.Parameters = params

			children = append(children, child)
		}

		actionsNotBuiltIn[i].Actions = children
	}

	// for all actions update its requirements from its children
	for i := range actionsNotBuiltIn {
		// copy requirements from child to action if enabled else ignore all its requirements
		if actionsNotBuiltIn[i].Enabled {
			for _, child := range actionsNotBuiltIn[i].Actions {
				if !child.Enabled { // if child is not enabled, ignore its requirements
					continue
				}

				// for each requirement of child, add it to parent if don't exists
				for _, cr := range child.Requirements {
					var found bool
					for _, pr := range actionsNotBuiltIn[i].Requirements {
						if pr.Type == cr.Type && pr.Value == cr.Value {
							found = true
							break
						}
					}
					if !found {
						actionsNotBuiltIn[i].Requirements = append(actionsNotBuiltIn[i].Requirements, cr)
					}
				}
			}
		} else {
			actionsNotBuiltIn[i].Requirements = make([]sdk.Requirement, 0)
		}
	}

	return nil
}

type edgeAggregator func(gorp.SqlExecutor, ...*actionEdge) error

func aggregateEdgeParameters(db gorp.SqlExecutor, es ...*actionEdge) error {
	ps, err := getEdgeParametersByEdgeIDs(db, actionEdgesToIDs(es))
	if err != nil {
		return err
	}

	m := make(map[int64][]actionEdgeParameter)
	for i := range ps {
		if _, ok := m[ps[i].ActionEdgeID]; !ok {
			m[ps[i].ActionEdgeID] = make([]actionEdgeParameter, 0)
		}
		m[ps[i].ActionEdgeID] = append(m[ps[i].ActionEdgeID], ps[i])
	}
	for i := range es {
		if ps, ok := m[es[i].ID]; ok {
			es[i].Parameters = ps
		}
	}

	return nil
}

func aggregateEdgeChildren(db gorp.SqlExecutor, es ...*actionEdge) error {
	children, err := loadAllByIDs(db, actionEdgesToChildIDs(es))
	if err != nil {
		return err
	}

	m := make(map[int64]sdk.Action, len(children))
	for i := range children {
		m[children[i].ID] = children[i]
	}

	for i := range es {
		if child, ok := m[es[i].ChildID]; ok {
			es[i].Child = &child
		}
	}

	return nil
}

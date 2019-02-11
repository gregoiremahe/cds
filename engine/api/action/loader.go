package action

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

// LoadAllTypeBuiltInOrDefault actions from database.
func LoadAllTypeBuiltInOrDefault(db gorp.SqlExecutor, ags ...actionAggregator) ([]sdk.Action, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM action
    WHERE (type = $1 OR type = $2)
    ORDER BY name
  `).Args(sdk.BuiltinAction, sdk.DefaultAction)
	return getAll(db, query, append([]actionAggregator{
		aggregateActionRequirements,
		aggregateActionParameters,
		aggregateActionChildren,
	}, ags...)...)
}

// LoadAllTypeBuiltInOrDefaultByGroupIDs actions from database.
func LoadAllTypeBuiltInOrDefaultByGroupIDs(db gorp.SqlExecutor, groupIDs []int64, ags ...actionAggregator) ([]sdk.Action, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM action
    WHERE (type = $1 OR type = $2) AND group_id = ANY(string_to_array($3, ',')::int[]
    ORDER BY name
  `).Args(sdk.BuiltinAction, sdk.DefaultAction, gorpmapping.IDsToQueryString(groupIDs))
	return getAll(db, query, append([]actionAggregator{
		aggregateActionRequirements,
		aggregateActionParameters,
		aggregateActionChildren,
	}, ags...)...)
}

func loadAllTypeBuiltInOrDefaultByIDsAndGroupIDs(db gorp.SqlExecutor, ids []int64, groupIDs []int64, ags ...actionAggregator) ([]sdk.Action, error) {
	query := gorpmapping.NewQuery(`
  SELECT *
  FROM action
  WHERE (type = $1 OR type = $2) AND id = ANY(string_to_array($3, ',')::int[]) AND group_id = ANY(string_to_array($4, ',')::int[])
`).Args(
		sdk.DefaultAction,
		sdk.BuiltinAction,
		gorpmapping.IDsToQueryString(ids),
		gorpmapping.IDsToQueryString(groupIDs),
	)
	return getAll(db, query,
		aggregateActionChildren,
	)
}

// loadAllByIDs retrieves in database actions with given ids.
func loadAllByIDs(db gorp.SqlExecutor, ids []int64) ([]sdk.Action, error) {
	query := gorpmapping.NewQuery("SELECT * FROM action WHERE action.id = ANY(string_to_array($1, ',')::int[])").
		Args(gorpmapping.IDsToQueryString(ids))
	return getAll(db, query,
		aggregateActionRequirements,
		aggregateActionParameters,
		aggregateActionChildren)
}

// LoadTypeBuiltInOrDefaultByName returns a action from database for given name.
func LoadTypeBuiltInOrDefaultByName(db gorp.SqlExecutor, name string) (*sdk.Action, error) {
	query := gorpmapping.NewQuery(
		"SELECT * FROM action WHERE (type = $1 OR type = $2) AND lower(action.name) = lower($3)",
	).Args(sdk.BuiltinAction, sdk.DefaultAction, name)
	return get(db, query,
		aggregateActionRequirements,
		aggregateActionParameters,
		aggregateActionChildren)
}

// LoadTypeBuiltInOrDefaultByNameAndGroupID returns an action from database with given name and group id.
func LoadTypeBuiltInOrDefaultByNameAndGroupID(db gorp.SqlExecutor, name string, groupID int64) (*sdk.Action, error) {
	query := gorpmapping.NewQuery(
		"SELECT * FROM action WHERE (type = $1 OR type = $2) AND lower(name) = lower($3) AND group_id = $4",
	).Args(sdk.BuiltinAction, sdk.DefaultAction, name, groupID)
	return get(db, query,
		aggregateActionRequirements,
		aggregateActionParameters,
		aggregateActionChildren)
}

// LoadTypePluginByName returns a action from database for given name.
func LoadTypePluginByName(db gorp.SqlExecutor, name string) (*sdk.Action, error) {
	query := gorpmapping.NewQuery(
		"SELECT * FROM action WHERE type = $1 AND lower(action.name) = lower($2)",
	).Args(sdk.PluginAction, name)
	return get(db, query,
		aggregateActionRequirements,
		aggregateActionParameters,
		aggregateActionChildren)
}

// LoadByID retrieves in database the action with given id.
func LoadByID(db gorp.SqlExecutor, id int64) (*sdk.Action, error) {
	query := gorpmapping.NewQuery("SELECT * FROM action WHERE action.id = $1").Args(id)
	return get(db, query,
		aggregateActionRequirements,
		aggregateActionParameters,
		aggregateActionChildren)
}

// loadEdgesByParentIDs retrieves in database all action edges for given parent ids.
func loadEdgesByParentIDs(db gorp.SqlExecutor, parentIDs []int64) ([]actionEdge, error) {
	query := gorpmapping.NewQuery(
		"SELECT * FROM action_edge WHERE parent_id = ANY(string_to_array($1, ',')::int[]) ORDER BY exec_order ASC",
	).Args(gorpmapping.IDsToQueryString(parentIDs))
	return getEdges(db, query,
		aggregateEdgeParameters,
		aggregateEdgeChildren)
}

package action

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

// LoadAll actions from database.
func LoadAll(db gorp.SqlExecutor) ([]sdk.Action, error) {
	query := gorpmapping.NewQuery("SELECT * FROM action WHERE public = true ORDER BY name")
	return getAll(db, query,
		aggregateActionRequirements,
		aggregateActionParameters,
		aggregateActionChildren)
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

// LoadPublicByName returns a public action from database for given name.
func LoadPublicByName(db gorp.SqlExecutor, name string) (*sdk.Action, error) {
	query := gorpmapping.NewQuery("SELECT * FROM action WHERE lower(action.name) = lower($1) AND public = true").
		Args(name)
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

package api

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	yaml "gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/log"
)

const (
	contextAction contextKey = iota
)

func (api *API) middlewareAction(needAdmin bool) func(ctx context.Context, w http.ResponseWriter, r *http.Request) (context.Context, error) {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) (context.Context, error) {
		// try to get action for given path that match user's groups with/without admin grants
		vars := mux.Vars(r)

		groupName := vars["groupName"]
		actionName := vars["actionName"]

		if groupName == "" || actionName == "" {
			return nil, sdk.WrapError(sdk.ErrWrongRequest, "invalid given group or action name")
		}

		u := deprecatedGetUser(ctx)

		// check that group exists
		g, err := group.LoadGroup(api.mustDB(), groupName)
		if err != nil {
			return nil, err
		}

		if needAdmin {
			if err := group.CheckUserIsGroupAdmin(g, u); err != nil {
				return nil, err
			}
		} else {
			if err := group.CheckUserIsGroupMember(g, u); err != nil {
				return nil, err
			}
		}

		a, err := action.LoadTypeBuiltInOrDefaultByNameAndGroupID(api.mustDB(), actionName, g.ID)
		if err != nil {
			return nil, err
		}

		return context.WithValue(ctx, contextAction, a), nil
	}
}

func getAction(c context.Context) *sdk.Action {
	i := c.Value(contextAction)
	if i == nil {
		return nil
	}
	a, ok := i.(*sdk.Action)
	if !ok {
		return nil
	}
	return a
}

func (api *API) getActionsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		u := deprecatedGetUser(ctx)

		var as []sdk.Action
		var err error
		if u.Admin {
			as, err = action.LoadAllTypeBuiltInOrDefault(
				api.mustDB(),
				group.AggregateOnAction,
			)
		} else {
			as, err = action.LoadAllTypeBuiltInOrDefaultByGroupIDs(
				api.mustDB(),
				append(sdk.GroupsToIDs(u.Groups), group.SharedInfraGroup.ID),
				group.AggregateOnAction,
			)
		}
		if err != nil {
			return err
		}

		return service.WriteJSON(w, as, http.StatusOK)
	}
}

func (api *API) postActionHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var data sdk.Action
		if err := service.UnmarshalBody(r, &data); err != nil {
			return err
		}
		if err := data.IsValid(); err != nil {
			return err
		}

		// check that the group exists and user is admin for group id
		grp, err := group.LoadGroupByID(api.mustDB(), *data.GroupID)
		if err != nil {
			return err
		}

		u := deprecatedGetUser(ctx)

		if err := group.CheckUserIsGroupAdmin(grp, u); err != nil {
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback()

		// check that no action already exists for same group/name
		current, err := action.LoadTypeBuiltInOrDefaultByNameAndGroupID(tx, data.Name, grp.ID)
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNoAction) {
			return err
		}
		if current != nil {
			return sdk.NewErrorFrom(sdk.ErrAlreadyExist, "an action already exists for given name on this group")
		}

		// only default action can be posted or updated
		data.Type = sdk.DefaultAction

		// only action from action's group and shared.infra can be used as child
		if err := action.CheckChildrenForGroupIDs(tx, &data, []int64{group.SharedInfraGroup.ID, grp.ID}); err != nil {
			return err
		}

		// inserts action and components
		if err := action.Insert(tx, &data); err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		event.PublishActionAdd(data, u)

		if err := group.AggregateOnAction(api.mustDB(), &data); err != nil {
			return err
		}

		return service.WriteJSON(w, data, http.StatusOK)
	}
}

func (api *API) getActionHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		ctx, err := api.middlewareAction(false)(ctx, w, r)
		if err != nil {
			return err
		}

		a := getAction(ctx)

		if err := group.AggregateOnAction(api.mustDB(), a); err != nil {
			return err
		}

		return service.WriteJSON(w, a, http.StatusOK)
	}
}

func (api *API) putActionHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		ctx, err := api.middlewareAction(true)(ctx, w, r)
		if err != nil {
			return err
		}

		a := getAction(ctx)

		var data sdk.Action
		if err := service.UnmarshalBody(r, &data); err != nil {
			return err
		}
		if err := data.IsValid(); err != nil {
			return err
		}

		// check that the group exists and user is admin for group id
		grp, err := group.LoadGroupByID(api.mustDB(), *data.GroupID)
		if err != nil {
			return err
		}

		// TODO in case of group change, we need to check if current action is used

		u := deprecatedGetUser(ctx)

		if err := group.CheckUserIsGroupAdmin(grp, u); err != nil {
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot begin transaction")
		}
		defer tx.Rollback()

		// only default action can be posted or updated
		data.ID = a.ID
		data.Type = sdk.DefaultAction

		// only action from action's group and shared.infra can be used as child
		if err := action.CheckChildrenForGroupIDsWithLoop(tx, &data, []int64{group.SharedInfraGroup.ID, grp.ID}); err != nil {
			return err
		}

		if err = action.UpdateActionDB(tx, &data, deprecatedGetUser(ctx).ID); err != nil {
			return sdk.WrapError(err, "cannot update action")
		}

		if err = tx.Commit(); err != nil {
			return sdk.WrapError(err, "cannot commit transaction")
		}

		event.PublishActionUpdate(*a, data, deprecatedGetUser(ctx))

		return service.WriteJSON(w, data, http.StatusOK)
	}
}

func (api *API) getActionAuditHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		actionID, err := requestVarInt(r, "actionID")
		if err != nil {
			return err
		}

		a, err := action.LoadAuditAction(api.mustDB(), actionID, true)
		if err != nil {
			return sdk.WrapError(err, "cannot load audit for action %d", actionID)
		}

		return service.WriteJSON(w, a, http.StatusOK)
	}
}

func (api *API) getActionUsageHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		ctx, err := api.middlewareAction(true)(ctx, w, r)
		if err != nil {
			return err
		}

		a := getAction(ctx)

		pus, err := action.GetPipelineUsages(api.mustDB(), a.ID)
		if err != nil {
			return err
		}
		aus, err := action.GetActionUsages(api.mustDB(), a.ID)
		if err != nil {
			return err
		}

		return service.WriteJSON(w, action.Usage{
			Pipelines: pus,
			Actions:   aus,
		}, http.StatusOK)
	}
}

func (api *API) deleteActionHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get action name in URL
		vars := mux.Vars(r)
		name := vars["actionName"]

		a, errLoad := action.LoadTypeBuiltInOrDefaultByName(api.mustDB(), name)
		if errLoad != nil {
			if !sdk.ErrorIs(errLoad, sdk.ErrNoAction) {
				log.Warning("deleteAction> Cannot load action %s: %T %s", name, errLoad, errLoad)
			}
			return errLoad
		}

		used, errUsed := action.Used(api.mustDB(), a.ID)
		if errUsed != nil {
			return errUsed
		}
		if used {
			return sdk.WrapError(sdk.ErrForbidden, "deleteAction> Cannot delete action %s: used in pipelines", name)
		}

		tx, errbegin := api.mustDB().Begin()
		if errbegin != nil {
			return sdk.WrapError(errbegin, "deleteAction> Cannot start transaction")
		}
		defer tx.Rollback()

		if err := action.DeleteAction(tx, a.ID, deprecatedGetUser(ctx).ID); err != nil {
			return sdk.WrapError(err, "Cannot delete action %s", name)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		event.PublishActionDelete(*a, deprecatedGetUser(ctx))

		return nil
	}
}

func (api *API) getActionExportHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		name := vars["actionName"]

		format := FormString(r, "format")
		if format == "" {
			format = "yaml"
		}

		f, err := exportentities.GetFormat(format)
		if err != nil {
			return err
		}

		if _, err := action.Export(api.mustDB(), name, f, w); err != nil {
			return sdk.WithStack(err)
		}

		w.Header().Add("Content-Type", exportentities.GetContentType(f))
		return nil
	}
}

// importActionHandler insert OR update an existing action.
func (api *API) importActionHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var a *sdk.Action

		data, errRead := ioutil.ReadAll(r.Body)
		if errRead != nil {
			return errRead
		}
		defer r.Body.Close()

		contentType := r.Header.Get("Content-Type")
		if contentType == "" {
			contentType = http.DetectContentType(data)
		}

		var ea = new(exportentities.Action)
		var errapp error
		switch contentType {
		case "application/json":
			errapp = json.Unmarshal(data, ea)
		case "application/x-yaml", "text/x-yam":
			errapp = yaml.Unmarshal(data, ea)
		default:
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "unsupported content-type: %s", contentType)
		}

		if errapp != nil {
			return sdk.NewError(sdk.ErrWrongRequest, errapp)
		}

		a, errapp = ea.Action()
		if errapp != nil {
			return sdk.NewError(sdk.ErrWrongRequest, errapp)
		}

		if a == nil {
			return sdk.WithStack(sdk.ErrWrongRequest)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback()

		// check if action exists
		exist := false
		existingAction, errload := action.LoadTypeBuiltInOrDefaultByName(tx, a.Name)
		if errload == nil {
			exist = true
			a.ID = existingAction.ID
		}

		// http code status
		var code int

		// update or Insert the action
		if exist {
			if err := action.UpdateActionDB(tx, a, deprecatedGetUser(ctx).ID); err != nil {
				return err
			}
			code = http.StatusOK
		} else {
			a.Type = sdk.DefaultAction
			if err := action.Insert(tx, a); err != nil {
				return err
			}
			code = http.StatusCreated
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		if exist {
			event.PublishActionUpdate(*existingAction, *a, deprecatedGetUser(ctx))
		} else {
			event.PublishActionAdd(*a, deprecatedGetUser(ctx))
		}

		return service.WriteJSON(w, a, code)
	}
}

func (api *API) getActionsRequirements() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		rs, err := action.GetRequirementsDistinctBinary(api.mustDB())
		if err != nil {
			return sdk.WrapError(err, "cannot load action requirements")
		}

		return service.WriteJSON(w, rs, http.StatusOK)
	}
}

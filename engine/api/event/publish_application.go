package event

import (
	"fmt"
	"time"

	"github.com/fatih/structs"

	"github.com/ovh/cds/sdk"
)

// PublishApplicationEvent publish application event
func publishApplicationEvent(payload interface{}, key, appName string, u *sdk.User) {
	event := sdk.Event{
		Timestamp:       time.Now(),
		Hostname:        hostname,
		CDSName:         cdsname,
		EventType:       fmt.Sprintf("%T", payload),
		Payload:         structs.Map(payload),
		ProjectKey:      key,
		ApplicationName: appName,
	}
	if u != nil {
		event.Username = u.Username
		event.UserMail = u.Email
	}
	publishEvent(event)
}

// PublishAddApplication publishes an event for the creation of the given application
func PublishAddApplication(projKey string, app sdk.Application, u *sdk.User) {
	e := sdk.EventApplicationAdd{
		Application: app,
	}
	publishApplicationEvent(e, projKey, app.Name, u)
}

// PublishUpdateApplication publishes an event for the update of the given application
func PublishUpdateApplication(projKey string, app sdk.Application, oldApp sdk.Application, u *sdk.User) {
	e := sdk.EventApplicationUpdate{
		NewMetadata:           app.Metadata,
		NewRepositoryStrategy: app.RepositoryStrategy,
		NewName:               app.Name,
		OldMetadata:           oldApp.Metadata,
		OldRepositoryStrategy: oldApp.RepositoryStrategy,
		OldName:               oldApp.Name,
	}
	publishApplicationEvent(e, projKey, app.Name, u)
}

// PublishDeleteApplication publishes an event for the deletion of the given application
func PublishDeleteApplication(projKey string, app sdk.Application, u *sdk.User) {
	e := sdk.EventApplicationDelete{}
	publishApplicationEvent(e, projKey, app.Name, u)
}

// PublishAddVariableApplication publishes an event when adding a new variable
func PublishAddVariableApplication(projKey string, app sdk.Application, v sdk.Variable, u *sdk.User) {
	if sdk.NeedPlaceholder(v.Type) {
		v.Value = sdk.PasswordPlaceholder
	}
	e := sdk.EventApplicationVariableAdd{
		Variable: v,
	}
	publishApplicationEvent(e, projKey, app.Name, u)
}

// PublishUpdateVariableApplication publishes an event when updating a variable
func PublishUpdateVariableApplication(projKey string, app sdk.Application, v sdk.Variable, vOld sdk.Variable, u *sdk.User) {
	e := sdk.EventApplicationVariableUpdate{
		OldVariable: vOld,
		NewVariable: v,
	}
	publishApplicationEvent(e, projKey, app.Name, u)
}

// PublishDeleteVariableApplication publishes an event when deleting a new variable
func PublishDeleteVariableApplication(projKey string, app sdk.Application, v sdk.Variable, u *sdk.User) {
	e := sdk.EventApplicationVariableDelete{
		Variable: v,
	}
	publishApplicationEvent(e, projKey, app.Name, u)
}

func PublishApplicationKeyAdd(projKey string, app sdk.Application, k sdk.ApplicationKey, u *sdk.User) {
	e := sdk.EventApplicationKeyAdd{
		Key: k,
	}
	publishApplicationEvent(e, projKey, app.Name, u)
}

func PublishApplicationKeyDelete(projKey string, app sdk.Application, k sdk.ApplicationKey, u *sdk.User) {
	e := sdk.EventApplicationKeyDelete{
		Key: k,
	}
	publishApplicationEvent(e, projKey, app.Name, u)
}

// PublishApplicationRepositoryAdd publishes an envet when adding a repository to an application
func PublishApplicationRepositoryAdd(projKey string, app sdk.Application, u *sdk.User) {
	e := sdk.EventApplicationRepositoryAdd{
		VCSServer:  app.VCSServer,
		Repository: app.RepositoryFullname,
	}
	publishApplicationEvent(e, projKey, app.Name, u)
}

// PublishApplicationRepositoryDelete publishes an envet when deleting a repository from an application
func PublishApplicationRepositoryDelete(projKey string, appName string, vcsServer string, repository string, u *sdk.User) {
	e := sdk.EventApplicationRepositoryDelete{
		VCSServer:  vcsServer,
		Repository: repository,
	}
	publishApplicationEvent(e, projKey, appName, u)
}

// PublishApplicationVulnerabilityUpdate publishes an event when updating a vulnerability
func PublishApplicationVulnerabilityUpdate(projKey string, appName string, oldV sdk.Vulnerability, newV sdk.Vulnerability, u *sdk.User) {
	e := sdk.EventApplicationVulnerabilityUpdate{
		OldVulnerability: oldV,
		NewVulnerability: newV,
	}
	publishApplicationEvent(e, projKey, appName, u)
}

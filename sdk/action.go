package sdk

import (
	"time"
)

// Action is the base element of CDS pipeline
type Action struct {
	ID          int64  `json:"id" yaml:"-" db:"id"`
	Name        string `json:"name" cli:"name,key" db:"name"`
	Type        string `json:"type" yaml:"-" cli:"type" db:"type"`
	Description string `json:"description" yaml:"desc,omitempty" db:"description"`
	Enabled     bool   `json:"enabled" yaml:"-" db:"enabled"`
	Deprecated  bool   `json:"deprecated" yaml:"-" db:"deprecated"`
	// aggregates from action_edge
	StepName       string `json:"step_name,omitempty" yaml:"step_name,omitempty" cli:"step_name" db:"-"`
	Optional       bool   `json:"optional" yaml:"-" db:"-"`
	AlwaysExecuted bool   `json:"always_executed" yaml:"-" db:"-"`
	// aggregates
	Requirements []Requirement `json:"requirements" db:"-"`
	Parameters   []Parameter   `json:"parameters" db:"-"`
	Actions      []Action      `json:"actions" yaml:"actions,omitempty" db:"-"`
	LastModified int64         `json:"last_modified" cli:"modified" db:"-"`
}

// ActionSummary is the light representation of an action for CDS event
type ActionSummary struct {
	Name     string `json:"name"`
	StepName string `json:"step_name"`
}

// ToSummary returns an ActionSummary from an Action
func (a Action) ToSummary() ActionSummary {
	return ActionSummary{
		Name:     a.Name,
		StepName: a.StepName,
	}
}

// ActionAudit Audit on action
type ActionAudit struct {
	ActionID   int64     `json:"action_id"`
	User       User      `json:"user"`
	Change     string    `json:"change"`
	Versionned time.Time `json:"versionned"`
	Action     Action    `json:"action"`
}

// Action type
const (
	DefaultAction = "Default"
	BuiltinAction = "Builtin"
	PluginAction  = "Plugin"
	JoinedAction  = "Joined"
)

// Builtin Action
const (
	ScriptAction              = "Script"
	JUnitAction               = "JUnit"
	CoverageAction            = "Coverage"
	GitCloneAction            = "GitClone"
	GitTagAction              = "GitTag"
	ReleaseAction             = "Release"
	CheckoutApplicationAction = "CheckoutApplication"
	DeployApplicationAction   = "DeployApplication"

	DefaultGitCloneParameterTagValue = "{{.git.tag}}"
)

// NewAction instanciate a new Action
func NewAction(name string) *Action {
	a := &Action{
		Name:    name,
		Enabled: true,
	}
	return a
}

// Parameter add given parameter to Action
func (a *Action) Parameter(p Parameter) *Action {
	a.Parameters = append(a.Parameters, p)
	return a
}

// Add takes an action that will be executed when current action is executed
func (a *Action) Add(child Action) *Action {
	a.Actions = append(a.Actions, child)
	return a
}

// NewScriptAction setup a new Action object with all attribute ok for script action
func NewScriptAction(content string) Action {
	var a Action

	a.Name = ScriptAction
	a.Type = BuiltinAction
	a.Enabled = true
	a.Parameters = append(a.Parameters, Parameter{Name: "script", Value: content})
	return a
}

// ActionsToIDs returns ids for given actions list.
func ActionsToIDs(as []*Action) []int64 {
	var ids []int64
	for i := range as {
		ids = append(ids, as[i].ID)
	}
	return ids
}

// ActionsFilterNotTypes returns a list of actions filtered by types.
func ActionsFilterNotTypes(as []*Action, ts ...string) []*Action {
	var f []*Action
	for i := range as {
		for j := range ts {
			if as[i].Type != ts[j] {
				f = append(f, as[i])
			}
		}
	}
	return f
}

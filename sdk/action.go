package sdk

import (
	"time"
)

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

// NewScriptAction setup a new Action object with all attribute ok for script action
func NewScriptAction(content string) Action {
	var a Action

	a.Name = ScriptAction
	a.Type = BuiltinAction
	a.Enabled = true
	a.Parameters = append(a.Parameters, Parameter{Name: "script", Value: content})
	return a
}

// Action is the base element of CDS pipeline
type Action struct {
	ID          int64  `json:"id" yaml:"-" db:"id"`
	GroupID     *int64 `json:"group_id" yaml:"-" db:"group_id"`
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
	Group        *Group        `json:"group" db:"-"`
}

// IsValid returns action validity.
func (a Action) IsValid() error {
	if a.GroupID == nil || *a.GroupID == 0 {
		return NewErrorFrom(ErrWrongRequest, "invalid group id for action")
	}
	if a.Name == "" {
		return NewErrorFrom(ErrWrongRequest, "invalid name for action")
	}

	for i := range a.Parameters {
		if err := a.Parameters[i].IsValid(); err != nil {
			return err
		}
	}

	for i := range a.Actions {
		if a.Actions[i].ID == 0 {
			return NewErrorFrom(ErrWrongRequest, "invalid action id for child")
		}
		for j := range a.Actions[i].Parameters {
			if err := a.Actions[i].Parameters[j].IsValid(); err != nil {
				return err
			}
		}
	}

	return nil
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

// ToUniqueChildrenIDs returns distinct children ids for given action.
func (a Action) ToUniqueChildrenIDs() []int64 {
	mChildrenIDs := make(map[int64]struct{}, len(a.Actions))
	for i := range a.Actions {
		mChildrenIDs[a.Actions[i].ID] = struct{}{}
	}
	childrenIDs := make([]int64, len(mChildrenIDs))
	i := 0
	for id := range mChildrenIDs {
		childrenIDs[i] = id
		i++
	}
	return childrenIDs
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

// ActionsToIDs returns ids for given actions list.
func ActionsToIDs(as []*Action) []int64 {
	ids := make([]int64, len(as))
	for i := range as {
		ids[i] = as[i].ID
	}
	return ids
}

// ActionsToGroupIDs returns group ids for given actions list.
func ActionsToGroupIDs(as []*Action) []int64 {
	ids := make([]int64, len(as))
	for i := range as {
		if as[i].GroupID != nil {
			ids[i] = *as[i].GroupID
		}
	}
	return ids
}

// ActionsFilterNotTypes returns a list of actions filtered by types.
func ActionsFilterNotTypes(as []*Action, ts ...string) []*Action {
	f := make([]*Action, 0, len(as))
	for i := range as {
		for j := range ts {
			if as[i].Type != ts[j] {
				f = append(f, as[i])
			}
		}
	}
	return f
}

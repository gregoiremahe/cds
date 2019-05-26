package gitlab

import (
	"context"
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/xanzy/go-gitlab"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"

	"time"
    "math/rand"
)

type statusData struct {
	status       string
	branchName   string
	url          string
	desc         string
	repoFullName string
	hash         string
}

var statusCache = []statusData{}

func getGitlabStateFromStatus(s string) gitlab.BuildStateValue {
	switch s {
	case sdk.StatusWaiting.String():
		log.Trace("vcs.gitlab.getGitlabStateFromStatus> status sdk.StatusWaiting --> gitlab.Pending")
		return gitlab.Pending
	case sdk.StatusChecking.String():
		log.Trace("vcs.gitlab.getGitlabStateFromStatus> status sdk.StatusChecking --> gitlab.Pending")
		return gitlab.Pending
	case sdk.StatusBuilding.String():
		log.Trace("vcs.gitlab.getGitlabStateFromStatus> status sdk.StatusBuilding --> gitlab.Running")
		return gitlab.Running
	case sdk.StatusSuccess.String():
		log.Trace("vcs.gitlab.getGitlabStateFromStatus> status sdk.StatusSuccess --> gitlab.Success")
		return gitlab.Success
	case sdk.StatusFail.String():
		log.Trace("vcs.gitlab.getGitlabStateFromStatus> status sdk.StatusFail --> gitlab.Failed")
		return gitlab.Failed
	case sdk.StatusDisabled.String():
		log.Trace("vcs.gitlab.getGitlabStateFromStatus> status sdk.StatusDisabled --> gitlab.Canceled")
		return gitlab.Canceled
	case sdk.StatusNeverBuilt.String():
		log.Trace("vcs.gitlab.getGitlabStateFromStatus> status sdk.StatusNeverBuilt --> gitlab.Canceled")
		return gitlab.Canceled
	case sdk.StatusUnknown.String():
		log.Trace("vcs.gitlab.getGitlabStateFromStatus> status sdk.StatusUnknown --> gitlab.Failed")
		return gitlab.Failed
	case sdk.StatusSkipped.String():
		log.Trace("vcs.gitlab.getGitlabStateFromStatus> status sdk.StatusSkipped --> gitlab.Canceled")
		return gitlab.Canceled
	}
	log.Trace("vcs.gitlab.getGitlabStateFromStatus> status sdk unknown case --> gitlab.Failed")
	return gitlab.Failed
}

//SetStatus set build status on Gitlab
func (c *gitlabClient) SetStatus(ctx context.Context, event sdk.Event) error {
	log.Trace("vcs.gitlab.SetStatus> doing")
	if c.disableStatus {
		log.Warning("disableStatus.SetStatus>  ⚠ Gitlab statuses are disabled")
		return nil
	}

	var data statusData
	var err error
	switch event.EventType {
	case fmt.Sprintf("%T", sdk.EventRunWorkflowNode{}):
		data, err = processWorkflowNodeRunEvent(event, c.uiURL)
	default:
		log.Debug("gitlabClient.SetStatus> Unknown event %v", event)
		return nil
	}

	if err != nil {
		return sdk.WrapError(err, "cannot process event %v", event)
	}

	if c.disableStatusDetail {
		data.url = ""
	}

	cds := "CDS"
	opt := &gitlab.SetCommitStatusOptions{
		Name:        &cds,
		Context:     &cds,
		State:       getGitlabStateFromStatus(data.status),
		Ref:         &data.branchName,
		TargetURL:   &data.url,
		Description: &data.desc,
	}

	val, _, err := c.client.Commits.GetCommitStatuses(data.repoFullName, data.hash, nil)
	if err != nil {
		return sdk.WrapError(err, "unable to get commit statuses - repo:%s hash:%s", data.repoFullName, data.hash)
	}
	found := false

	log.Trace("vcs.gitlab.SetStatus> -- Starting for val -- ")
	for _, s := range val {

		// Comparing statuses on gitlab and CDS one
		sameRequest := s.TargetURL == *opt.TargetURL && // Comparing TargetURL as there is the workflow run number inside
			s.Status == string(opt.State) && // Comparing Status to avoid duplicate entries
			s.Ref == *opt.Ref && // Comparing branches name
			s.SHA == data.hash && // Comparing commit SHA to match the right commit
			s.Name == *opt.Name && // Comparing app name (CDS)
			s.Description == *opt.Description // Comparing Description as there are the pipelines names inside

		log.Trace("vcs.gitlab.SetStatus> --- Comparing GitLab status VS CDS one")
		log.Trace("vcs.gitlab.SetStatus> TargetURL: %s  %s", s.TargetURL, *opt.TargetURL)
		log.Trace("vcs.gitlab.SetStatus> Status: %s  %s", s.Status, string(opt.State))
		log.Trace("vcs.gitlab.SetStatus> Description: %s  %s", s.Description, *opt.Description)
		log.Trace("vcs.gitlab.SetStatus> Name: %s  %s", s.Name, *opt.Name)
		log.Trace("vcs.gitlab.SetStatus> Ref: %s  %s", s.Ref, *opt.Ref)
		log.Trace("vcs.gitlab.SetStatus> SHA: %s  %s", s.SHA, data.hash)
		log.Trace("vcs.gitlab.SetStatus> CreatedAt: %s", s.CreatedAt)
		log.Trace("vcs.gitlab.SetStatus> StartedAt: %s", s.StartedAt)
		log.Trace("vcs.gitlab.SetStatus> FinishedAt: %s", s.FinishedAt)
		log.Trace("vcs.gitlab.SetStatus> repoFullName: %s", data.repoFullName)
		log.Trace("vcs.gitlab.SetStatus> -------")

		if sameRequest {
			log.Debug("gitlabClient.SetStatus> Duplicate commit status, ignoring request - repo:%s hash:%s", data.repoFullName, data.hash)
			found = true
			break
		}
	}

	time.Sleep(time.Duration(rand.Intn(10) * 100) * time.Millisecond)
	sameStatus := false
	if len(statusCache) > 0 {
		lastStatus := statusCache[len(statusCache) - 1]

		// Comparing last status sent on gitlab and CDS one
		sameStatus = lastStatus.url == *opt.TargetURL && // Comparing TargetURL as there is the workflow run number inside
			lastStatus.status == string(opt.State) && // Comparing Status to avoid duplicate entries
			lastStatus.branchName == *opt.Ref && // Comparing branches name
			lastStatus.hash == data.hash && // Comparing commit SHA to match the right commit
			lastStatus.desc == *opt.Description // Comparing Description as there are the pipelines names inside
	}

	if !found && !sameStatus {
		log.Trace("vcs.gitlab.SetStatus> statusCache before sending: %s", statusCache)
		if _, _, err := c.client.Commits.SetCommitStatus(data.repoFullName, data.hash, opt); err != nil {
			return sdk.WrapError(err, "cannot process event %v - repo:%s hash:%s", event, data.repoFullName, data.hash)
		}
		statusCache = append(statusCache, statusData{
			status:			string(opt.State),
			branchName:		*opt.Ref,
			url:			*opt.TargetURL,
			desc:			*opt.Description,
			repoFullName:	data.repoFullName,
			hash:			data.hash,
		})
		log.Trace("vcs.gitlab.SetStatus> statusCache after sending: %s", statusCache)
	}
	return nil
}

func (c *gitlabClient) ListStatuses(ctx context.Context, repo string, ref string) ([]sdk.VCSCommitStatus, error) {
	log.Trace("vcs.gitlab.ListStatuses> doing")
	ss, _, err := c.client.Commits.GetCommitStatuses(repo, ref, nil)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to get commit statuses hash:%s", ref)
	}

	vcsStatuses := []sdk.VCSCommitStatus{}
	for _, s := range ss {
		if !strings.HasPrefix(s.Description, "CDS/") {
			continue
		}
		/*log.Trace("vcs.gitlab.ListStatuses> TargetURL: %s", s.TargetURL)
		log.Trace("vcs.gitlab.ListStatuses> Status: %s", s.Status)
		log.Trace("vcs.gitlab.ListStatuses> Description: %s", s.Description)
		log.Trace("vcs.gitlab.ListStatuses> Name: %s", s.Name)
		log.Trace("vcs.gitlab.ListStatuses> Ref: %s", s.Ref)
		log.Trace("vcs.gitlab.ListStatuses> SHA: %s", s.SHA)
		log.Trace("vcs.gitlab.ListStatuses> CreatedAt: %s", s.CreatedAt)
		log.Trace("vcs.gitlab.ListStatuses> StartedAt: %s", s.StartedAt)
		log.Trace("vcs.gitlab.ListStatuses> FinishedAt: %s", s.FinishedAt)
		log.Trace("vcs.gitlab.ListStatuses> repo: %s", repo)*/

		vcsStatuses = append(vcsStatuses, sdk.VCSCommitStatus{
			CreatedAt:  *s.CreatedAt,
			Decription: s.Description,
			Ref:        ref,
			State:      processGitlabState(*s),
		})
	}
	log.Trace("vcs.gitlab.ListStatuses> vcsStatuses: %s", vcsStatuses)

	return vcsStatuses, nil
}

func processGitlabState(s gitlab.CommitStatus) string {
	switch s.Status {
	case string(gitlab.Success):
		log.Trace("vcs.gitlab.processGitlabState> status gitlab.Success --> sdk.StatusSuccess")
		return sdk.StatusSuccess.String()
	case string(gitlab.Failed):
		log.Trace("vcs.gitlab.processGitlabState> status gitlab.Failed --> sdk.StatusFail")
		return sdk.StatusFail.String()
	case string(gitlab.Canceled):
		log.Trace("vcs.gitlab.processGitlabState> status gitlab.Canceled --> sdk.StatusSkipped")
		return sdk.StatusSkipped.String()
	default:
		log.Trace("vcs.gitlab.processGitlabState> status default --> sdk.StatusBuilding")
		return sdk.StatusBuilding.String()
	}
}

func processWorkflowNodeRunEvent(event sdk.Event, uiURL string) (statusData, error) {
	log.Trace("vcs.gitlab.processWorkflowNodeRunEvent> doing")

	data := statusData{}
	var eventNR sdk.EventRunWorkflowNode
	if err := mapstructure.Decode(event.Payload, &eventNR); err != nil {
		return data, sdk.WrapError(err, "cannot read payload")
	}

	data.url = fmt.Sprintf("%s/project/%s/workflow/%s/run/%d",
		uiURL,
		event.ProjectKey,
		event.WorkflowName,
		eventNR.Number,
	)
	log.Trace("vcs.gitlab.processWorkflowNodeRunEvent> data.url: %s", data.url)

	data.desc = sdk.VCSCommitStatusDescription(event.ProjectKey, event.WorkflowName, eventNR)
	data.hash = eventNR.Hash
	data.repoFullName = eventNR.RepositoryFullName
	data.status = eventNR.Status
	data.branchName = eventNR.BranchName
	log.Trace("vcs.gitlab.processWorkflowNodeRunEvent> data.desc: %s", data.desc)
	log.Trace("vcs.gitlab.processWorkflowNodeRunEvent> data.hash: %s", data.hash)
	log.Trace("vcs.gitlab.processWorkflowNodeRunEvent> data.repoFullName: %s", data.repoFullName)
	log.Trace("vcs.gitlab.processWorkflowNodeRunEvent> data.status: %s", data.status)
	log.Trace("vcs.gitlab.processWorkflowNodeRunEvent> data.branchName: %s", data.branchName)
	return data, nil
}

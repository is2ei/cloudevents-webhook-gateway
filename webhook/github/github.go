package github

import (
	"fmt"
	"net/http"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v01"
	"github.com/google/go-github/github"
)

type Parser struct {
	secret []byte
}

func NewParser(secret string) *Parser {
	return &Parser{
		secret: []byte(secret),
	}
}

func (p *Parser) Parse(req *http.Request) (*cloudevents.Event, error) {
	payload, err := github.ValidatePayload(req, p.secret)
	if err != nil {
		return nil, err
	}

	webHookType := github.WebHookType(req)
	event, err := github.ParseWebHook(webHookType, payload)
	if err != nil {
		return nil, err
	}

	var (
		action    string
		source    string
		eventType string
	)

	switch event := event.(type) {
	case *github.CheckRunEvent:
		source = event.CheckRun.GetHTMLURL()
		action = event.GetAction()
	case *github.CheckSuiteEvent:
		source = event.CheckSuite.GetURL()
	case *github.CommitCommentEvent:
		source = event.Comment.GetHTMLURL()
		action = event.GetAction()
	case *github.CreateEvent:
		source = event.Repo.GetHTMLURL()
	case *github.DeleteEvent:
		source = event.Repo.GetHTMLURL()
	case *github.DeploymentEvent:
		source = event.Deployment.GetURL()
	case *github.DeploymentStatusEvent:
		source = event.Deployment.GetURL()
	case *github.ForkEvent:
		source = event.Forkee.GetHTMLURL()
	case *github.GollumEvent:
		source = event.Repo.GetHTMLURL()
	case *github.InstallationEvent:
		source = event.Installation.GetHTMLURL()
		action = event.GetAction()
	case *github.InstallationRepositoriesEvent:
		source = event.Installation.GetHTMLURL()
		action = event.GetAction()
	case *github.IssueCommentEvent:
		source = event.Comment.GetHTMLURL()
		action = event.GetAction()
	case *github.IssuesEvent:
		source = event.Issue.GetHTMLURL()
		action = event.GetAction()
	case *github.LabelEvent:
		source = event.Label.GetURL()
		action = event.GetAction()
	case *github.MarketplacePurchaseEvent:
		source = event.Sender.GetHTMLURL()
		action = event.GetAction()
	case *github.MemberEvent:
		source = event.Member.GetHTMLURL()
		action = event.GetAction()
	case *github.MembershipEvent:
		source = event.Team.GetURL()
		action = event.GetAction()
	case *github.MilestoneEvent:
		source = event.Milestone.GetHTMLURL()
		action = event.GetAction()
	case *github.OrganizationEvent:
		source = event.Organization.GetHTMLURL()
		action = event.GetAction()
	case *github.OrgBlockEvent:
		source = event.Organization.GetHTMLURL()
		action = event.GetAction()
	case *github.PageBuildEvent:
		source = event.Build.GetURL()
	case *github.ProjectCardEvent:
		source = event.ProjectCard.GetURL()
		action = event.GetAction()
	case *github.ProjectColumnEvent:
		source = event.ProjectColumn.GetURL()
		action = event.GetAction()
	case *github.ProjectEvent:
		source = event.Project.GetHTMLURL()
		action = event.GetAction()
	case *github.PublicEvent:
		source = event.Repo.GetHTMLURL()
	case *github.PullRequestReviewCommentEvent:
		source = event.Comment.GetHTMLURL()
		action = event.GetAction()
	case *github.PullRequestReviewEvent:
		source = event.Review.GetHTMLURL()
		action = event.GetAction()
	case *github.PullRequestEvent:
		source = event.PullRequest.GetURL()
		action = event.GetAction()
	case *github.PushEvent:
		source = event.Repo.GetHTMLURL()
	case *github.RepositoryEvent:
		source = event.Repo.GetHTMLURL()
		action = event.GetAction()
	case *github.ReleaseEvent:
		source = event.Release.GetHTMLURL()
		action = event.GetAction()
	case *github.StatusEvent:
		source = event.Commit.GetHTMLURL()
	case *github.TeamEvent:
		source = event.Team.GetURL()
		action = event.GetAction()
	case *github.TeamAddEvent:
		source = event.Team.GetURL()
	case *github.WatchEvent:
		source = event.Repo.GetHTMLURL()
		action = event.GetAction()
	}

	if action == "" {
		eventType = fmt.Sprintf("com.github.%s", webHookType)
	} else {
		eventType = fmt.Sprintf("com.github.%s.%s", webHookType, action)
	}

	t := time.Now()

	ce := &cloudevents.Event{
		EventID:          github.DeliveryID(req),
		EventTime:        &t,
		EventType:        eventType,
		EventTypeVersion: "3",
		Source:           source,
		SchemaURL:        "https://developer.github.com/v3/activity/events/types",
		ContentType:      "application/json",
	}

	return ce, nil
}
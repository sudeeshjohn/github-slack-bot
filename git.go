package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/go-github/v45/github"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"
)

type LabelAction struct {
	IssueNumber int
	Action      string
}
type IssueAction struct {
	Number      int
	State       string
	Action      string
	UserName    string
	LastUpdated string
	Labels      []string
}
type MemberAction struct {
	UserName string
	Action   string
	Team     string
}
type TeamAction struct {
	Action string
}
type GithubActions struct {
	Organization string
	Repository   string
	Issue        *IssueAction
	Member       *MemberAction
	Label        *LabelAction
	Team         *TeamAction
}

var supportedIssueStates = []string{"open", "closed", "assigned", "unassigned", "assignedto", "createdby"}
var supportedIssueActions = []string{"get", "list"}
var supportedTeamActions = []string{"list"}
var supportedLabelActions = []string{"get", "list"}
var supportedLabelOptions = []string{"issue"}
var supportedIssueOptions = []string{"user", "noupdatesince", "labels"}
var supportedMemberActions = []string{"get", "add"}
var supportedMemberOptions = []string{"team"}
var ExcludeTeamName = []string{"legacy-team", "admin"}
var statesNeedsUserName = []string{"assignedto", "createdby"}

//var availableStates = []string{"open", "closed", "assigned", "unassigned"}

func getGitClient() (*github.Client, context.Context, error) {
	// Initilizing git client
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_OAUTH_TOKEN")},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	if len(os.Getenv("GITHUB_ENTERPRISE_URL")) > 0 {
		enterpriseURL, err := url.Parse(os.Getenv("GITHUB_ENTERPRISE_URL"))
		if err != nil {
			return nil, nil, fmt.Errorf("unable update new github client custom URL, Error: %s", err)
		}
		client.BaseURL = enterpriseURL
	}
	return client, ctx, nil
}

func validateUser(userName string, organization string) (bool, string, error) {

	var err error
	var reason string
	client, ctx, err := getGitClient()
	if err != nil {
		return false, "Internal Error", fmt.Errorf("unable update New github client, Error: %s", err)
	} else {
		_, res, err := client.Users.Get(ctx, userName)
		if err != nil {
			reason = "Unknown User"
			log.Info().Msg(fmt.Sprintf("Unable to find user, in the org `%s`", organization))
			return false, reason, err
		}
		if res.StatusCode != 200 {
			log.Info().Msg(fmt.Sprintf("Unable to find user, in the org `%s`", organization))
			reason = "Unknown User"
			return false, reason, err
		} else {
			log.Info().Msg(fmt.Sprintf("User found"))
			return true, "", nil
		}
	}
}

func validateIssue(issueNumber int, repository string, organization string) (bool, string, error) {

	var err error
	var reason string
	client, ctx, err := getGitClient()
	if err != nil {
		return false, "Internal Error", fmt.Errorf("unable update New github client, Error: %s", err)
	} else {
		_, res, err := client.Issues.Get(ctx, organization, repository, issueNumber)
		if err != nil {
			reason = "Unknown Issue"
			log.Info().Msg(fmt.Sprintf("Unable to find issue `%d`, in the repo `%s`", issueNumber, repository))
			return false, reason, err
		}
		if res.StatusCode != 200 {
			log.Info().Msg(fmt.Sprintf("Unable to find issue `%d`, in the repo `%s`", issueNumber, repository))
			reason = "Unknown User"
			return false, reason, err
		} else {
			log.Info().Msg(fmt.Sprintf("User found: %d", issueNumber))
			return true, "", nil
		}
	}
}

func (g GithubActions) userGet() (bool, string, error) {

	var err error
	var reason string
	client, ctx, err := getGitClient()
	if err != nil {
		return false, "Internal Error", fmt.Errorf("unable update New github client, Error: %s", err)
	}
	user, res, err := client.Users.Get(ctx, g.Member.UserName)
	if err != nil {
		reason = "Unknown User"
		return false, reason, fmt.Errorf("unable to find user. Error:%s", err)
	}
	if res.StatusCode != 200 {
		return false, "Unknown User", fmt.Errorf("unable to find user. Error: %s", err)
	}
	return true, fmt.Sprintf("Login:\t*<%s|%s>*\n \nName:\t*`%s`*\nEmail:\t*`%s`*\nID:\t*`%d`*\nPublic Repos:\t*`%d`*\n", *user.HTMLURL, *user.Login, *user.Name, *user.Email, *user.ID, *user.PublicRepos), nil
}

func (g GithubActions) validateTeam() (bool, string, error) {
	client, ctx, err := getGitClient()
	if err != nil {
		log.Info().Msg(fmt.Sprintf("Unable update New github client, Error: %s", err))
		return false, "Internal Error", err
	} else {
		_, res, err := client.Teams.GetTeamBySlug(ctx, g.Organization, g.Member.Team)
		if err != nil {
			log.Info().Msg(fmt.Sprintf("Unable to find the Team `%s`", g.Member.Team))
			return false, "Unknown Team", err
		}
		if res.StatusCode != 200 {
			log.Info().Msg(fmt.Sprintf("unable to find the team `%s`, you have to give a valid team name\n", g.Member.Team))
			return false, "Unknown Team", err
		} else {
			log.Info().Msg("Team found")
			return true, "", nil
		}

	}
}
func (g GithubActions) actOnMember() (bool, string, error) {
	stat, err := g.validateInputs()
	if !stat {
		return false, "Unknown Options", fmt.Errorf("unknown inputs. Error:%s", err)
	}
	switch {
	case g.Member.Action == "add":

		_, msg, err := g.addMember()
		if err != nil {
			return false, msg, err
		}
		return true, msg, nil
	case g.Member.Action == "get":
		_, msg, err := g.userGet()
		if err != nil {
			return false, msg, err
		}
		return true, msg, nil
	default:
		return false, "", fmt.Errorf("unknown Action")
	}
	//return true, "", nil
}

func (g GithubActions) actOnLabel() (bool, []string, string, error) {
	var labelList []string
	var message string
	var err error
	var stat bool
	stat, message, err = g.validateRepoAndOrg()
	if !stat {
		return false, labelList, message, fmt.Errorf("invalid org/repo. Error: %s", err)
	}
	stat, err = g.validateInputs()
	if !stat {
		return false, labelList, "Unknown Options", fmt.Errorf("unknown inputs, Error:%s", err)
	}
	switch {
	case g.Label.Action == "get":
		if g.Label.IssueNumber > 0 {
			stat, message, err = validateIssue(g.Label.IssueNumber, g.Repository, g.Organization)
			if !stat {
				return false, labelList, message, fmt.Errorf("invalid issue. Error: %s", err)
			}
		}
		labelList, message, err = ListLabelsByIssue(g.Organization, g.Repository, g.Label.IssueNumber)
		return true, labelList, message, nil

	case g.Label.Action == "list":
		labelList, message, err = ListLabels(g.Organization, g.Repository)
		if err != nil {
			return false, labelList, message, err
		}
		return true, labelList, message, nil
	default:
		return false, labelList, "", fmt.Errorf("unknown Action")
	}
}

func (g GithubActions) actOnIssue() (bool, []string, string, error) {
	var issueList []string
	var message string
	var err error
	var since time.Time
	stat, message, err := g.validateRepoAndOrg()
	if !stat {
		return false, issueList, message, fmt.Errorf("invalid org/repo. Error: %s", err)
	}
	switch {
	case g.Issue.Action == "list":
		var lstOpt *github.ListOptions
		lstOpt = &github.ListOptions{Page: 1, PerPage: 100}
		var options *github.IssueListByRepoOptions
		options = &github.IssueListByRepoOptions{ListOptions: *lstOpt}

		if len(g.Issue.LastUpdated) != 0 {
			since, _ = time.Parse("2006-01-02", g.Issue.LastUpdated)
		}
		stat, err = g.validateInputs()
		if !stat {
			return false, issueList, "Unknown Options", fmt.Errorf("unknown inputs, Error:%s", err)
		}
		if len(g.Issue.UserName) > 0 {
			stat, message, err = validateUser(g.Issue.UserName, g.Organization)
			if !stat {
				return false, issueList, message, fmt.Errorf("invalid user name. Error: %s", err)
			}
		}
		if len(g.Issue.LastUpdated) > 0 {
			if !isDateValue(g.Issue.LastUpdated) {
				return false, issueList, message, fmt.Errorf("Invalid date string. Date string must be in 2021-06-01 format")
			}
		}
		if len(g.Issue.Labels) > 0 {
			options.Labels = g.Issue.Labels
		}
		switch {
		case g.Issue.State == "createdby":
			options.State = "open"
			options.Creator = g.Issue.UserName
			issueList, message, err = issuesListByRepo(options, g.Organization, g.Repository, since)
			return true, issueList, message, nil
		case g.Issue.State == "assignedto":
			options.State = "open"
			options.Assignee = g.Issue.UserName
			issueList, message, err = issuesListByRepo(options, g.Organization, g.Repository, since)
			return true, issueList, message, nil
		case g.Issue.State == "open":
			options.State = "open"
			issueList, message, err = issuesListByRepo(options, g.Organization, g.Repository, since)
			if len(g.Issue.UserName) != 0 {
				message = message + fmt.Sprintf("\t(ignored input user=%s)", g.Issue.UserName)
			}
			return true, issueList, message, nil
		case g.Issue.State == "closed":
			options.State = "closed"
			issueList, message, err = issuesListByRepo(options, g.Organization, g.Repository, since)
			if len(g.Issue.UserName) != 0 {
				message = message + fmt.Sprintf("\t(ignored input user=%s)", g.Issue.UserName)
			}
			return true, issueList, message, nil
		case g.Issue.State == "assigned":
			options.State = "open"
			options.Assignee = "*"
			issueList, message, err = issuesListByRepo(options, g.Organization, g.Repository, since)
			if len(g.Issue.UserName) != 0 {
				message = message + fmt.Sprintf("\t(ignored input user=%s)", g.Issue.UserName)
			}
			return true, issueList, message, nil
		case g.Issue.State == "unassigned":
			options.State = "open"
			options.Assignee = "none"
			issueList, message, err = issuesListByRepo(options, g.Organization, g.Repository, since)
			if len(g.Issue.UserName) != 0 {
				message = message + fmt.Sprintf("\t(ignored input user=%s)", g.Issue.UserName)
			}
			return true, issueList, message, nil
		}

	case g.Issue.Action == "get":
		if g.Issue.Number == 0 {
			return false, issueList, "", fmt.Errorf("invalid issue number given. issue %s take a issue number as input", g.Issue.Action)
		}
		_, message, err = g.issueGet()
		if err != nil {
			return false, issueList, message, err
		}
		return true, issueList, message, nil
	default:
		return false, issueList, "", fmt.Errorf("unknown Action")
	}
	return true, issueList, "", nil
}

func ListTeams(Org string) ([]string, string, error) {
	var teamList []string
	var message string
	lstopt := &github.ListOptions{
		Page:    1,
		PerPage: 100,
	}
	client, ctx, err := getGitClient()
	if err != nil {
		return nil, "Internal Error", fmt.Errorf("unable update New github client, Error: %s", err)
	}
	for {
		teams, resp, err := client.Teams.ListTeams(ctx, Org, lstopt)
		if err != nil {
			return nil, "Internal Error", fmt.Errorf("getting team failed, Error: %s", err)
		}
		if len(teams) > 0 {
			for _, team := range teams {
				if !contains(ExcludeTeamName, *team.Name) {
					teamList = append(teamList, fmt.Sprintf("*<%s|%s>*\t*`%s`*\n", team.GetURL(), *team.Name, *team.Description))
				}
			}
		} else {
			message = fmt.Sprintf("No team found in Org: `\"%s\"`", Org)
			return teamList, message, fmt.Errorf("no teams found, Error: %s", err)
		}
		if resp.NextPage == 0 {
			break
		}
		lstopt.Page = resp.NextPage
	}
	return teamList, fmt.Sprintf("*`%d team/s found`*", len(teamList)), err
}

func (g GithubActions) actOnTeam() (bool, []string, string, error) {
	var teamList []string
	var message string
	var err error
	stat, message, err := g.validateRepoAndOrg()
	if !stat {
		return false, teamList, message, fmt.Errorf("invalid org/repo. Error: %s", err)
	}
	switch {
	case g.Team.Action == "list":
		stat, err = g.validateInputs()
		if !stat {
			return false, teamList, "Unknown Options", fmt.Errorf("unknown inputs, Error:%s", err)
		}
		teamList, message, err = ListTeams(g.Organization)
		return true, teamList, message, nil

	default:
		return false, teamList, "", fmt.Errorf("unknown Action")
	}
	return true, teamList, "", nil
}

func ListLabelsByIssue(Org string, Repo string, Issue int) ([]string, string, error) {
	var labelList []string
	var message string
	lstopt := &github.ListOptions{
		Page:    1,
		PerPage: 100,
	}
	client, ctx, err := getGitClient()
	if err != nil {
		return nil, "Internal Error", fmt.Errorf("unable update New github client, Error: %s", err)
	}
	for {
		labels, resp, err := client.Issues.ListLabelsByIssue(ctx, Org, Repo, Issue, lstopt)
		if err != nil {
			return nil, "Internal Error", fmt.Errorf("getting labels failed, Error: %s", err)
		}
		if len(labels) > 0 {
			for _, label := range labels {
				labelList = append(labelList, fmt.Sprintf("*<%s|%s>*\n", label.GetURL(), label.GetName()))
			}
		} else {
			message = fmt.Sprintf("No labels found for issue : `\"%d\"`", Issue)
			return labelList, message, fmt.Errorf("no lables found, Error: %s", err)
		}
		if resp.NextPage == 0 {
			break
		}
		lstopt.Page = resp.NextPage
	}
	return labelList, fmt.Sprintf("*`%d labels found for issue %d`*", len(labelList), Issue), err
}
func ListLabels(Org string, Repo string) ([]string, string, error) {
	var labelList []string
	var message string
	lstopt := &github.ListOptions{
		Page:    1,
		PerPage: 100,
	}
	client, ctx, err := getGitClient()
	if err != nil {
		return nil, "Internal Error", fmt.Errorf("unable update New github client, Error: %s", err)
	}
	for {
		labels, resp, err := client.Issues.ListLabels(ctx, Org, Repo, lstopt)
		if err != nil {
			return nil, "Internal Error", fmt.Errorf("getting labels failed, Error: %s", err)
		}
		if len(labels) > 0 {
			for _, label := range labels {
				labelList = append(labelList, fmt.Sprintf("*<%s|%s>*\t*`%s`*\n", label.GetURL(), label.GetName(), label.GetDescription()))
			}
		} else {
			message = fmt.Sprintf("No labels found in Org: `\"%s\"`, Repo: `\"%s\"`", Org, Repo)
			return labelList, message, fmt.Errorf("no lables found, Error: %s", err)
		}
		if resp.NextPage == 0 {
			break
		}
		lstopt.Page = resp.NextPage
	}
	return labelList, fmt.Sprintf("*`%d labels found`*", len(labelList)), err
}
func issuesListByRepo(opts *github.IssueListByRepoOptions, Org string, Repo string, since time.Time) ([]string, string, error) {
	var issueList []string
	var message string
	client, ctx, err := getGitClient()
	if err != nil {
		return nil, "Internal Error", fmt.Errorf("unable update New github client, Error: %s", err)
	}
	for {
		issues, resp, err := client.Issues.ListByRepo(ctx, Org, Repo, opts)
		if err != nil {
			return nil, "Internal Error", fmt.Errorf("getting issues failed, Error: %s", err)
		}
		if len(issues) > 0 {
			for _, issu := range issues {
				if !issu.IsPullRequest() {
					//if issu.Assignee.GetLogin() == opts.Assignee { //added newly
					if !since.IsZero() {
						if issu.UpdatedAt.Sub(since) < 0 {
							issueList = append(issueList, fmt.Sprintf("*<%s|%d>*\t*`%s`*\n", issu.GetHTMLURL(), *issu.Number, *issu.Title))
						}
					} else {
						issueList = append(issueList, fmt.Sprintf("*<%s|%d>*\t*`%s`*\n", issu.GetHTMLURL(), *issu.Number, *issu.Title))
					}
					//}
				}
			}
		} else {
			message = fmt.Sprintf("No issues found in Org: `\"%s\"`, Repo: `\"%s\"`", Org, Repo)
			return issueList, message, fmt.Errorf("no issues found, Error: %s", err)
		}
		if resp.NextPage == 0 {
			break
		}
		opts.ListOptions.Page = resp.NextPage
	}
	return issueList, fmt.Sprintf("*`%d issues found`*", len(issueList)), err
}

func (g GithubActions) issueGet() (bool, string, error) {
	var err error
	var allAssignees string
	client, ctx, err := getGitClient()
	if err != nil {
		return false, "internal Error", fmt.Errorf("failed to create git client. Error %s", err)
	}
	stat, msg, err := g.validateRepoAndOrg()
	if !stat {
		return false, msg, fmt.Errorf("unable to get org/repo. Error: %s", err)
	}
	issue, resp, err := client.Issues.Get(ctx, g.Organization, g.Repository, g.Issue.Number)
	if err != nil {
		return false, "No such issue", fmt.Errorf("unable to find  issue. Error:%s", err)
	}
	if resp.StatusCode != 200 {
		return false, "Unknown issue", fmt.Errorf("unable to find issue. Error: %s", err)
	}
	for _, assig := range issue.Assignees {
		allAssignees = allAssignees + fmt.Sprintf("%s, ", assig.GetLogin())
	}
	return true, fmt.Sprintf("*ID*:\t*<%s|%d>*\n*Title*: *`%s`*\n*Assigned To:*\t*`%s`*\n*State:*\t*`%s`*\n*Last update on:*\t*`%s`*\n*Body:*\n \t%s\n", issue.GetHTMLURL(), *issue.Number, *issue.Title, allAssignees, *issue.State, issue.GetUpdatedAt(), issue.GetBody()), nil
}

func (g GithubActions) addMember() (bool, string, error) {
	var orgStatus string
	var teamStatus string
	orgStatus, err := g.OrganizationsEditOrgMembership()
	if err != nil {
		return false, "", fmt.Errorf("user `%s` failed to add to the organization `%s`. Error: %s", g.Member.UserName, g.Organization, err)
	}
	if orgStatus == "User Added" || orgStatus == "Already A Member" {
		teamStatus, err = g.TeamsAddTeamMembershipBySlug()
		if err != nil {
			return false, "", fmt.Errorf("user `%s` failed to add to the team `%s`. Error: %s", g.Member.UserName, g.Member.Team, err)
		}
		if teamStatus == "User Added" {
			return true, fmt.Sprintf("user `%s` added to team `%s` ", g.Member.UserName, g.Member.Team), nil
		} else if teamStatus == "Already A Member" {
			return true, fmt.Sprintf("user `%s` is already a member of team `%s`", g.Member.UserName, g.Member.Team), nil
		} else {
			return false, "", fmt.Errorf("user `%`s failed to add to the team `%s`. Erro: %s", g.Member.UserName, g.Member.Team, err)
		}
	} else {
		return false, "", fmt.Errorf("user `%s` failed to add to the team `%s`. Error: %s", g.Member.UserName, g.Member.Team, err)
	}
}
func (g GithubActions) OrganizationsEditOrgMembership() (string, error) {
	var status string
	stat, _ := g.checkIfUserAlreadyMemberOfOrg()
	if !stat {
		stat, reas, err := validateUser(g.Member.UserName, g.Organization)
		if err != nil {
			return reas, fmt.Errorf("unable to get the user details, Error: %s", err)
		}
		if stat {
			log.Debug().Msg(fmt.Sprintf("User %s is a valid user", g.Member.UserName))
			client, ctx, err := getGitClient()
			if err != nil {
				return "Internal Error", fmt.Errorf("unable update New github client, Error: %s", err)
			}
			_, resp, err := client.Organizations.EditOrgMembership(ctx, g.Member.UserName, g.Organization, nil)
			if err != nil {
				return "Failed to Add", fmt.Errorf("unable to add the user `%s`. Error: %s", g.Member.UserName, err)
			}
			if resp.StatusCode != 200 {
				return "Failed to Add", fmt.Errorf("unable to add user `%s` to the org `%s`", g.Member.UserName, g.Organization)
			} else {
				status = "User Added"
				log.Debug().Msg(fmt.Sprintf("User `%s` added to the Org `%s`", g.Member.UserName, g.Organization))
			}
		} else {
			return "Unknown User", fmt.Errorf("unable to add user `%s` to the org `%s`", g.Member.UserName, g.Organization)
		}
	} else {
		status = "Already A Member"
		log.Info().Msg("User already a member of org")
	}
	return status, nil
}

func (g GithubActions) TeamsAddTeamMembershipBySlug() (string, error) {
	var status string
	stat, _ := g.checkIfUserAlreadyMemberOfTeam()
	if !stat {
		stat, reas, err := validateUser(g.Member.UserName, g.Organization)
		if err != nil {
			return reas, fmt.Errorf("unable to get the User %s details, Error: %s", g.Member.UserName, err)
		}
		stat, reas, err = g.validateTeam()
		if err != nil {
			return reas, fmt.Errorf("unable to get the Team, %s details, Error: %s", g.Member.Team, err)
		}
		if stat {
			log.Debug().Msg(fmt.Sprintf("User %s is a valid user", g.Member.UserName))
			client, ctx, err := getGitClient()
			if err != nil {
				return "Internal Error", fmt.Errorf("unable update New github client, Error: %s", err)
			}
			_, resp, err := client.Teams.AddTeamMembershipBySlug(ctx, g.Organization, g.Member.Team, g.Member.UserName, nil)
			if err != nil {
				return "Failed to Add", fmt.Errorf("unable to add the user %s. Error: %s", g.Member.UserName, err)
			}
			if resp.StatusCode != 200 {
				fmt.Errorf("unable to add user %s to the Team %s", g.Member.UserName, g.Member.Team)
				status = "Failed to Add"
			} else {
				status = "User Added"
				log.Debug().Msg(fmt.Sprintf("User %s added to the OTeam %s", g.Member.UserName, g.Member.Team))
			}
		} else {
			status = "Unknown Team"
			return status, fmt.Errorf("unknown user %s", g.Member.Team)
		}
	} else {
		status = "Already A Member"
		log.Info().Msg("User already a member of Team")
	}
	return status, nil
}

func (g GithubActions) checkIfUserAlreadyMemberOfOrg() (bool, error) {
	//Organization Membership
	client, ctx, err := getGitClient()
	if err != nil {
		log.Info().Msg(fmt.Sprintf("Unable update New github client, Error: %s", err))
	} else {
		stat, _, err := client.Organizations.IsMember(ctx, g.Organization, g.Member.UserName)
		if err != nil {
			log.Info().Msg(fmt.Sprintf("Unable to get the user details. Error: %s", err))
		}
		if stat {
			log.Debug().Msg(fmt.Sprintf("User  %s is already a member of the Org %s", g.Member.UserName, g.Organization))
			return true, nil
		} else {
			log.Debug().Msg(fmt.Sprintf("User %s is not a member of the org %s", g.Member.UserName, g.Organization))
		}
	}
	return false, err
}

func (g GithubActions) checkIfUserAlreadyMemberOfTeam() (bool, error) {
	client, ctx, err := getGitClient()
	if err != nil {
		log.Info().Msg(fmt.Sprintf("Unable update New github client, Error: %s", err))
	} else {
		mem, rsp, _ := client.Teams.GetTeamMembershipBySlug(ctx, g.Organization, g.Member.Team, g.Member.UserName)
		/*if err != nil {
			log.Debug().Msg(fmt.Sprintf("Unable to get the team details. Error: %s", err))
		}*/
		if rsp.Response.StatusCode != 200 {
			log.Debug().Msg(fmt.Sprintf("User %s is not a member of the team %s", g.Member.UserName, g.Member.Team))
		} else {
			if *mem.State == "active" {
				log.Debug().Msg(fmt.Sprintf("User  %s is already a member of the team %s", g.Member.UserName, g.Member.Team))
				return true, nil
			} else if *mem.State == "pending" {
				log.Debug().Msg(fmt.Sprintf("There is already an invite sent to %s", g.Member.UserName))
			} else {
				log.Debug().Msg("Unknown status")
			}
		}
	}
	return false, err
}

func (g GithubActions) validateRepoAndOrg() (bool, string, error) {
	lstopt := &github.ListOptions{
		Page:    1,
		PerPage: 100,
	}
	opts := &github.RepositoryListByOrgOptions{ListOptions: *lstopt}
	var message string
	var err error
	client, ctx, err := getGitClient()
	if err != nil {
		log.Info().Msg(fmt.Sprintf("Unable update New github client, Error: %s", err))
		return false, "Internal Error", err
	}
	for {
		repoList, resp, err := client.Repositories.ListByOrg(ctx, g.Organization, opts)
		if err != nil {
			message = fmt.Sprintf("Unable to get details of Organization:`%s` and/or Repository: `%s`", g.Organization, g.Repository)
			log.Info().Msg(message)
			return false, message, err
		}
		log.Debug().Msg(fmt.Sprintf("Page numer: %d", opts.ListOptions.Page))
		if len(repoList) > 0 {
			for _, repo := range repoList {
				if *repo.Name == g.Repository && repo.Owner.GetLogin() == g.Organization {
					message = fmt.Sprintf("Organization: %s and the repository is %s\n", repo.Owner.GetLogin(), *repo.Name)
					log.Info().Msg(message)
					return true, message, nil
				}
				message = fmt.Sprintf("Unknown Organization:`%s`and/or the repository is `%s`", repo.Owner.GetLogin(), *repo.Name)
			}
		}
		log.Debug().Msg(fmt.Sprintf("Page numer: %d", resp.NextPage))
		if resp.NextPage == 0 {
			break
		}
		opts.ListOptions.Page = resp.NextPage
	}
	return false, message, err
}

func isDateValue(stringDate string) bool {
	_, err := time.Parse("2006-01-02", stringDate)
	return err == nil
}
func prettyPrint(i interface{}) string {
	var userDetails string
	var x map[string]interface{}
	jsonStr, _ := json.Marshal(i)
	json.Unmarshal([]byte(jsonStr), &x)
	for key, opt := range x {
		userDetails = userDetails + fmt.Sprintf("%v = %v \n", key, opt)

	}
	return SortString(userDetails)
}
func SortString(w string) string {
	s := strings.Split(w, "\n")
	sort.Strings(s)
	return strings.Join(s, "\n")
}

func (g GithubActions) validateInputs() (bool, error) {
	if g.Issue != nil {
		if len(g.Issue.UserName) == 0 && contains(statesNeedsUserName, g.Issue.State) {
			return false, fmt.Errorf("`%s` requires github user name as input", g.Issue.State)
		}
		/*if len(g.Issue.UserName) != 0 && contains(availableStates, g.Issue.State) {
			return false, fmt.Errorf("`%s` does not require github user name as input", g.Issue.State)
		}*/
		if g.Issue.Action == "get" && g.Issue.Number == 0 {
			return false, fmt.Errorf("`%s` expects an issue number as input", g.Issue.Action)
		}
	}
	if g.Member != nil {
		if len(g.Member.Team) == 0 && g.Member.Action == "add" {
			return false, fmt.Errorf("`%s` expects team name as input", g.Member.Action)
		}
	}
	return true, nil
}

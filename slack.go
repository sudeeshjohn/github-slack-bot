package main

import (
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/shomali11/slacker"
	"github.com/slack-go/slack"
	"os"
	"strconv"
	"strings"
)

type Bot struct {
	token string
}

func NewBot(token string) *Bot {
	return &Bot{
		token: token,
	}
}

func (b *Bot) Start() error {
	bot := slacker.NewClient(os.Getenv("SLACK_BOT_TOKEN"), os.Getenv("SLACK_APP_TOKEN"))

	bot.DefaultCommand(func(botCtx slacker.BotContext, request slacker.Request, response slacker.ResponseWriter) {
		err := response.Reply("unrecognized command, msg me `help` for a list of all commands")
		if err != nil {
			log.Info().Msg("unrecognized command, msg me `help` for a list of all commands")
		}
	})

	bot.Command("member <action> <user> <options>", &slacker.CommandDefinition{
		Description: fmt.Sprintf("Run the requested action %s on gihub user ", strings.Join(codeSlice(supportedMemberActions), ", ")),
		Example:     "member add sudeeshjohn team=storage",
		Handler: func(botCtx slacker.BotContext, request slacker.Request, response slacker.ResponseWriter) {
			var err error
			githubOrg := os.Getenv("GITHUB_ORG")
			githubRepo := os.Getenv("GITHUB_REPO")
			//user := botCtx.Event().User
			channel := botCtx.Event().Channel
			if !isDirectMessage(channel) {
				err := response.Reply("this command is only accepted via direct message")
				if err != nil {
					log.Info().Msg("this command is only accepted via direct message")
				}
				return
			}
			action, err := parseActions(request.StringParam("action", ""), supportedMemberActions)
			if err != nil {
				response.Reply(err.Error())
				return
			}
			if len(action) == 0 {
				response.Reply("you must specify what action need to be taken") //nolint:errcheck
				return
			}

			user := request.StringParam("user", "")
			log.Debug().Str("user", user).Msg("Received github id ")
			if len(user) == 0 || len(strings.Fields(user)) > 1 {
				response.Reply("You must specify a user") //nolint:errcheck
				return
			}
			params, err := parseOptions(request.StringParam("options", ""), supportedMemberOptions)
			if err != nil {
				response.Reply(err.Error())
				return
			}

			if len(params["team"]) > 0 {
				if contains(params["team"], "admin") || contains(params["team"], "Admin") {
					response.Reply("You are not prevailed to update `admin` team") //nolint:errcheck
					return
				}
			}
			/*if strings.EqualFold(params["team"], "admin") || strings.EqualFold(params["team"], "admins") {
				response.Reply("You are not prevailed to update `admin` team") //nolint:errcheck
				return
			}*/
			memAct := &MemberAction{
				UserName: user,
				Action:   action,
				Team:     strings.Join(params["team"], ","),
			}
			githubAct := GithubActions{
				Organization: githubOrg,
				Repository:   githubRepo,
				Issue:        nil,
				Member:       memAct,
			}
			status, msg, err := githubAct.actOnMember()
			if status {
				response.Reply(msg)
				return
			} else {
				response.Reply(err.Error())
			}
		},
	})

	bot.Command("issue <action>  <state_or_id> <options>", &slacker.CommandDefinition{
		Description: fmt.Sprintf("Run the requested action %s on gihub issues with different options like %s ", strings.Join(codeSlice(supportedIssueActions), ", "), strings.Join(codeSlice(supportedIssueOptions), ", ")),
		Example:     "1) issue get 234  2) issue list assignedto user=sudeeshjohn;noupdatesince=2022-07-01;labels=bug  3) issues list open noupdatesince=2022-07-01",
		Handler: func(botCtx slacker.BotContext, request slacker.Request, response slacker.ResponseWriter) {
			var err error
			var usr string
			var noupdate string
			githubOrg := os.Getenv("GITHUB_ORG")
			githubRepo := os.Getenv("GITHUB_REPO")
			//user := botCtx.Event().User
			channel := botCtx.Event().Channel
			attachments := []slack.Attachment{}
			if !isDirectMessage(channel) {
				err := response.Reply("this command is only accepted via direct message")
				if err != nil {
					log.Info().Msg("this command is only accepted via direct message")
				}
				return
			}
			action, err := parseActions(request.StringParam("action", ""), supportedIssueActions)
			if err != nil {
				response.Reply(err.Error())
				return
			}
			if len(action) == 0 {
				response.Reply("you must specify what action need to be taken")
				return
			}

			state, number, err := parseIssueState(request.StringParam("state_or_id", ""))
			if err != nil {
				response.Reply(err.Error())
				return
			}

			params, err := parseOptions(request.StringParam("options", ""), supportedIssueOptions)
			if err != nil {
				response.Reply(err.Error())
				return
			}
			if len(params["noupdatesince"]) != 0 {

				if !isDateValue(params["noupdatesince"][0]) {
					msg := "invalid date string, msg me `help` for example\""
					response.Reply(msg)
					return
				}
				noupdate = strings.Join(params["noupdatesince"], ",")
			}
			if len(params["user"]) > 0 {
				usr = strings.Join(params["user"], ",")
			}

			IssueAct := &IssueAction{
				Number:      number,
				State:       state,
				Action:      action,
				UserName:    usr,
				LastUpdated: noupdate,
				Labels:      params["labels"],
			}
			githubAct := GithubActions{
				Organization: githubOrg,
				Repository:   githubRepo,
				Issue:        IssueAct,
				Member:       nil,
			}

			status, issueList, msg, err := githubAct.actOnIssue()
			if status {
				if len(issueList) != 0 {
					var list string
					response.Reply(msg)
					count := 60
					for i := 0; i < len(issueList); i++ {
						attachments = []slack.Attachment{}
						if i < count {
							list = list + fmt.Sprintf(fmt.Sprintf(issueList[i]))

						} else {
							attachments = append(attachments, slack.Attachment{
								ID:   count,
								Text: list,
							})
							log.Debug().Msg(fmt.Sprintf("list : %s", list))
							response.Reply("", slacker.WithAttachments(attachments))
							count = count + 60
							list = ""
							i = i - 1
						}
					}
					attachments = append(attachments, slack.Attachment{
						ID:   count,
						Text: list,
					})
					log.Debug().Msg(fmt.Sprintf("list : %s", list))
					response.Reply("", slacker.WithAttachments(attachments))

				} else {
					attachments = []slack.Attachment{}

					attachments = append(attachments, slack.Attachment{
						ID:   1,
						Text: msg,
					})
					response.Reply("", slacker.WithAttachments(attachments))

				}
				return
			} else {
				response.Reply(err.Error())
			}
		},
	})

	bot.Command("labels <action>  <options>", &slacker.CommandDefinition{
		Description: fmt.Sprintf("Run the requested action %s on gihub labels with different options like %s ", strings.Join(codeSlice(supportedLabelActions), ", "), strings.Join(codeSlice(supportedLabelOptions), ", ")),
		Example:     "1) labels list  2) labels get 234",
		Handler: func(botCtx slacker.BotContext, request slacker.Request, response slacker.ResponseWriter) {
			var err error
			var issueNumber int
			githubOrg := os.Getenv("GITHUB_ORG")
			githubRepo := os.Getenv("GITHUB_REPO")
			//user := botCtx.Event().User
			channel := botCtx.Event().Channel
			attachments := []slack.Attachment{}
			if !isDirectMessage(channel) {
				err := response.Reply("this command is only accepted via direct message")
				if err != nil {
					log.Info().Msg("this command is only accepted via direct message")
				}
				return
			}
			action, err := parseActions(request.StringParam("action", ""), supportedLabelActions)
			if err != nil {
				response.Reply(err.Error())
				return
			}
			if len(action) == 0 {
				response.Reply("you must specify what action need to be taken")
				return
			}

			params, err := parseOptions(request.StringParam("options", ""), supportedLabelOptions)
			if err != nil {
				response.Reply(err.Error())
				return
			}
			if len(params["issue"]) != 0 {
				iss := strings.Join(params["issue"], ",")
				issueNumber, err = strconv.Atoi(iss)
				if err != nil {
					response.Reply(err.Error())
					return
				}
			}

			LabelAct := &LabelAction{
				IssueNumber: issueNumber,
				Action:      action,
			}
			githubAct := GithubActions{
				Organization: githubOrg,
				Repository:   githubRepo,
				Issue:        nil,
				Member:       nil,
				Label:        LabelAct,
			}

			status, labelList, msg, err := githubAct.actOnLabel()
			if status {
				if len(labelList) != 0 {
					var list string
					response.Reply(msg)
					count := 60
					for i := 0; i < len(labelList); i++ {
						attachments = []slack.Attachment{}
						if i < count {
							list = list + fmt.Sprintf(fmt.Sprintf(labelList[i]))

						} else {
							attachments = append(attachments, slack.Attachment{
								ID:   count,
								Text: list,
							})
							log.Debug().Msg(fmt.Sprintf("list : %s", list))
							response.Reply("", slacker.WithAttachments(attachments))
							count = count + 60
							list = ""
							i = i - 1
						}
					}
					attachments = append(attachments, slack.Attachment{
						ID:   count,
						Text: list,
					})
					log.Debug().Msg(fmt.Sprintf("list : %s", list))
					response.Reply("", slacker.WithAttachments(attachments))

				} else {
					attachments = []slack.Attachment{}

					attachments = append(attachments, slack.Attachment{
						ID:   1,
						Text: msg,
					})
					response.Reply("", slacker.WithAttachments(attachments))

				}
				return
			} else {
				response.Reply(err.Error())
			}
		},
	})

	bot.Command("version", &slacker.CommandDefinition{
		Description: "Report the version of the bot",
		Handler: func(botCtx slacker.BotContext, request slacker.Request, response slacker.ResponseWriter) {
			err := response.Reply(fmt.Sprintf("Running from https://github.com/sudeeshjohn/github-slack-bot"))
			if err != nil {
				log.Info().Msg("Unable to send the slack message")
			}
		},
	})

	return bot.Listen(context.Background())
}

func isDirectMessage(channel string) bool {
	return strings.HasPrefix(channel, "D")
}

func parseActions(action string, supportedActions []string) (string, error) {
	//params, err := paramsFromAnnotation(actions)
	if len(action) == 0 || len(strings.Fields(action)) > 1 {
		return "", fmt.Errorf("Action mustnot be empty or many")
	}
	if !contains(supportedActions, action) {
		return "", fmt.Errorf("Invalid Action")
	}
	act := strings.TrimSpace(action)
	return act, nil
}
func parseOptions(options string, supportedOptions []string) (map[string][]string, error) {
	params, err := paramsFromAnnotation(options)
	fmt.Printf("params from parse: %s", params)
	if err != nil {
		return nil, fmt.Errorf("options could not be parsed: %v", err)
	}
	for key, opt := range params {
		if len(opt) == 0 || len(key) == 0 {
			return nil, fmt.Errorf("option given not supported. msg me `help` for example")
		}
		switch {
		case contains(supportedOptions, key):
			if len(opt) == 0 {
				return nil, fmt.Errorf("empty %s is not supported", key)
			}
		default:
			return nil, fmt.Errorf("unrecognized option: %s", key)
		}
	}
	return params, nil
}
func paramsFromAnnotation(str string) (map[string][]string, error) {
	values := make(map[string][]string)
	var value string
	if len(str) == 0 {
		return values, nil
	}

	for _, part := range strings.Split(str, ";") {
		if len(part) == 0 {
			return nil, fmt.Errorf("parameter may not be empty")
		}
		parts := strings.SplitN(part, "=", 2)
		key := strings.TrimSpace(parts[0])
		if len(parts) > 1 {
			value = parts[1]
		}
		if len(key) == 0 {
			return nil, fmt.Errorf("parameter name may not be empty")
		}
		if len(parts) == 1 {
			values[key] = make([]string, 0)
			continue
		}
		values[key] = append(values[key], value)
	}
	return values, nil
}
func parseIssueState(stateOrID string) (string, int, error) {
	if len(stateOrID) == 0 || len(strings.Fields(stateOrID)) > 1 {
		return "", 0, fmt.Errorf("state/id mustnot be empty or many. msg me `help` for more information")
	}
	if !contains(supportedIssueStates, stateOrID) {
		if i, err := strconv.Atoi(stateOrID); err == nil {
			return "", i, nil
		}
		return "", 0, fmt.Errorf("invalid state or id: `%s`. msg me `help` for more information", stateOrID)
	}
	stat := strings.TrimSpace(stateOrID)
	return stat, 0, nil
}

func contains(arr []string, s string) bool {
	for _, item := range arr {
		if s == item {
			return true
		}
	}
	return false
}

func codeSlice(items []string) []string {
	code := make([]string, 0, len(items))
	for _, item := range items {
		code = append(code, fmt.Sprintf("`%s`", item))
	}
	return code
}

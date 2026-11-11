//go:generate mockery --name IssuesService

package gogitlab

import (
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type IssuesService interface {
	ListIssues(opt *gitlab.ListIssuesOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error)
	ListProjectIssues(pid interface{}, opt *gitlab.ListProjectIssuesOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error)
}

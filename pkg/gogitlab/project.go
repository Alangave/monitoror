//go:generate mockery --name ProjectService

package gogitlab

import (
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type ProjectService interface {
	GetProject(pid interface{}, opt *gitlab.GetProjectOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Project, *gitlab.Response, error)
}

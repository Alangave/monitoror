//go:generate mockery --name PipelinesService

package gogitlab

import (
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type PipelinesService interface {
	GetPipeline(pid interface{}, pipeline int64, options ...gitlab.RequestOptionFunc) (*gitlab.Pipeline, *gitlab.Response, error)
	ListProjectPipelines(pid interface{}, opt *gitlab.ListProjectPipelinesOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.PipelineInfo, *gitlab.Response, error)
}

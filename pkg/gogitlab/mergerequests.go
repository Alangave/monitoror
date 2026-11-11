//go:generate mockery --name MergeRequestsService

package gogitlab

import (
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type MergeRequestsService interface {
	GetMergeRequest(pid interface{}, mergeRequest int64, opt *gitlab.GetMergeRequestsOptions, options ...gitlab.RequestOptionFunc) (*gitlab.MergeRequest, *gitlab.Response, error)
	ListProjectMergeRequests(pid interface{}, opt *gitlab.ListProjectMergeRequestsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.BasicMergeRequest, *gitlab.Response, error)
	ListMergeRequestPipelines(pid interface{}, mergeRequest int64, options ...gitlab.RequestOptionFunc) ([]*gitlab.PipelineInfo, *gitlab.Response, error)
}

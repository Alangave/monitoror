package repository

import (
	"fmt"
	"net/http"
	"time"

	"github.com/monitoror/monitoror/monitorables/gitlab/api"
	"github.com/monitoror/monitoror/monitorables/gitlab/api/models"
	"github.com/monitoror/monitoror/monitorables/gitlab/config"
	"github.com/monitoror/monitoror/pkg/gogitlab"

	"github.com/AlekSi/pointer"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type (
	gitlabRepository struct {
		config *config.Gitlab

		issuesService        gogitlab.IssuesService
		pipelinesService     gogitlab.PipelinesService
		mergeRequestsService gogitlab.MergeRequestsService
		projectService       gogitlab.ProjectService
	}
)

func NewGitlabRepository(config *config.Gitlab) api.Repository {
	httpClient := &http.Client{
		Timeout: time.Duration(config.Timeout) * time.Millisecond,
	}
	gitlabAPIBaseURL := fmt.Sprintf("%s/api/v4", config.URL)

	git, err := gitlab.NewClient(config.Token, gitlab.WithBaseURL(gitlabAPIBaseURL), gitlab.WithHTTPClient(httpClient))
	if err != nil {
		// only when gitlabAPIBaseURL is not a valid URL
		panic(fmt.Sprintf("unable to setup Gitlab client\n. %v\n", err))
	}

	return &gitlabRepository{
		config: config,

		issuesService:        git.Issues,
		pipelinesService:     git.Pipelines,
		mergeRequestsService: git.MergeRequests,
		projectService:       git.Projects,
	}
}

func (gr *gitlabRepository) GetCountIssues(params *models.IssuesParams) (int, error) {
	var labels *gitlab.LabelOptions
	if len(params.Labels) > 0 {
		options := gitlab.LabelOptions(params.Labels)
		labels = &options
	}

	var authorID *int64
	if params.AuthorID != nil {
		authorIDValue := int64(*params.AuthorID)
		authorID = &authorIDValue
	}

	var assigneeID *gitlab.AssigneeIDValue
	if params.AssigneeID != nil {
		assigneeID = gitlab.AssigneeID(*params.AssigneeID)
	}

	var resp *gitlab.Response
	var err error
	if params.ProjectID != nil {
		listProjectIssueOption := &gitlab.ListProjectIssuesOptions{
			State:      params.State,
			Labels:     labels,
			Milestone:  params.Milestone,
			Scope:      params.Scope,
			Search:     params.Search,
			AuthorID:   authorID,
			AssigneeID: assigneeID,
		}

		_, resp, err = gr.issuesService.ListProjectIssues(*params.ProjectID, listProjectIssueOption)
		if err != nil {
			return 0, err
		}
	} else {
		listIssueOption := &gitlab.ListIssuesOptions{
			State:      params.State,
			Labels:     labels,
			Milestone:  params.Milestone,
			Scope:      params.Scope,
			Search:     params.Search,
			AuthorID:   authorID,
			AssigneeID: assigneeID,
		}

		_, resp, err = gr.issuesService.ListIssues(listIssueOption)
		if err != nil {
			return 0, err
		}
	}
	return int(resp.TotalItems), nil
}

func (gr *gitlabRepository) GetPipeline(projectID, pipelineID int) (*models.Pipeline, error) {
	gitlabPipeline, _, err := gr.pipelinesService.GetPipeline(int64(projectID), int64(pipelineID))
	if err != nil {
		return nil, err
	}

	pipeline := &models.Pipeline{
		ID:         int(gitlabPipeline.ID),
		Branch:     gitlabPipeline.Ref,
		Status:     gitlabPipeline.Status,
		StartedAt:  gitlabPipeline.StartedAt,
		FinishedAt: gitlabPipeline.FinishedAt,
	}

	if gitlabPipeline.User != nil {
		pipeline.Author.Name = gitlabPipeline.User.Name
		pipeline.Author.AvatarURL = gitlabPipeline.User.AvatarURL

		if pipeline.Author.Name == "" {
			pipeline.Author.Name = gitlabPipeline.User.Username
		}
	}

	return pipeline, nil
}

func (gr *gitlabRepository) GetPipelines(projectID int, ref string) ([]int, error) {
	var ids []int

	gitlabPipelines, _, err := gr.pipelinesService.ListProjectPipelines(int64(projectID), &gitlab.ListProjectPipelinesOptions{
		Ref:     &ref,
		OrderBy: pointer.ToString("id"),
		Sort:    pointer.ToString("desc"),
	})
	if err != nil {
		return nil, err
	}

	for _, pipeline := range gitlabPipelines {
		ids = append(ids, int(pipeline.ID))
	}

	return ids, nil
}

func (gr *gitlabRepository) GetMergeRequest(projectID, mergeRequestID int) (*models.MergeRequest, error) {
	gitlabMergeRequest, _, err := gr.mergeRequestsService.GetMergeRequest(int64(projectID), int64(mergeRequestID), &gitlab.GetMergeRequestsOptions{})
	if err != nil {
		return nil, err
	}

	return parseMergeRequest(&gitlabMergeRequest.BasicMergeRequest), nil
}

func (gr *gitlabRepository) GetMergeRequests(projectID int) ([]models.MergeRequest, error) {
	var mergeRequests []models.MergeRequest

	gitlabMergeRequests, _, err := gr.mergeRequestsService.ListProjectMergeRequests(int64(projectID), &gitlab.ListProjectMergeRequestsOptions{
		// If needed by users, use pagination.
		ListOptions: gitlab.ListOptions{
			Page:    1,
			PerPage: 100, // Maximum par_page allowed.
		},
		State: pointer.ToString("opened"),
	})
	if err != nil {
		return nil, err
	}

	for _, gitlabMergeRequest := range gitlabMergeRequests {
		mergeRequests = append(mergeRequests, *parseMergeRequest(gitlabMergeRequest))
	}

	return mergeRequests, nil
}

func (gr *gitlabRepository) GetMergeRequestPipelines(projectID int, mergeRequestID int) ([]int, error) {
	var ids []int

	gitlabPipelines, _, err := gr.mergeRequestsService.ListMergeRequestPipelines(int64(projectID), int64(mergeRequestID))
	if err != nil {
		return nil, err
	}

	for _, pipeline := range gitlabPipelines {
		ids = append(ids, int(pipeline.ID))
	}

	return ids, nil
}

func (gr *gitlabRepository) GetProject(projectID int) (*models.Project, error) {
	gitlabProject, _, err := gr.projectService.GetProject(int64(projectID), &gitlab.GetProjectOptions{})
	if err != nil {
		return nil, err
	}

	project := &models.Project{
		ID:         int(gitlabProject.ID),
		Owner:      gitlabProject.Namespace.Path,
		Repository: gitlabProject.Path,
	}

	return project, nil
}

func parseMergeRequest(gitlabMergeRequest *gitlab.BasicMergeRequest) *models.MergeRequest {
	mergeRequest := &models.MergeRequest{
		ID:              int(gitlabMergeRequest.IID),
		Title:           gitlabMergeRequest.Title,
		SourceProjectID: int(gitlabMergeRequest.SourceProjectID),
		SourceBranch:    gitlabMergeRequest.SourceBranch,
		CommitSHA:       gitlabMergeRequest.SHA,
	}

	if gitlabMergeRequest.Author != nil {
		mergeRequest.Author.Name = gitlabMergeRequest.Author.Name
		mergeRequest.Author.AvatarURL = gitlabMergeRequest.Author.AvatarURL

		if mergeRequest.Author.Name == "" {
			mergeRequest.Author.Name = gitlabMergeRequest.Author.Username
		}
	}

	return mergeRequest
}

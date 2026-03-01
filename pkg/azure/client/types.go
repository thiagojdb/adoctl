package client

type PullRequestStatus string

const (
	PullRequestStatusNotSet    PullRequestStatus = "notSet"
	PullRequestStatusActive    PullRequestStatus = "active"
	PullRequestStatusAbandoned PullRequestStatus = "abandoned"
	PullRequestStatusCompleted PullRequestStatus = "completed"
	PullRequestStatusAll       PullRequestStatus = "all"
)

type PullRequestAsyncStatus string

const (
	PullRequestAsyncStatusNotSet           PullRequestAsyncStatus = "notSet"
	PullRequestAsyncStatusQueued           PullRequestAsyncStatus = "queued"
	PullRequestAsyncStatusConflicts        PullRequestAsyncStatus = "conflicts"
	PullRequestAsyncStatusSucceeded        PullRequestAsyncStatus = "succeeded"
	PullRequestAsyncStatusRejectedByPolicy PullRequestAsyncStatus = "rejectedByPolicy"
	PullRequestAsyncStatusFailure          PullRequestAsyncStatus = "failure"
)

type GitPullRequestMergeStrategy string

const (
	GitPullRequestMergeStrategyNoFastForward GitPullRequestMergeStrategy = "noFastForward"
	GitPullRequestMergeStrategySquash        GitPullRequestMergeStrategy = "squash"
	GitPullRequestMergeStrategyRebase        GitPullRequestMergeStrategy = "rebase"
	GitPullRequestMergeStrategyRebaseMerge   GitPullRequestMergeStrategy = "rebaseMerge"
)

type IdentityRef struct {
	Id          string `json:"id,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	UniqueName  string `json:"uniqueName,omitempty"`
	Url         string `json:"url,omitempty"`
	ImageUrl    string `json:"imageUrl,omitempty"`
	Descriptor  string `json:"descriptor,omitempty"`
}

type GitRepository struct {
	Id            string      `json:"id,omitempty"`
	Name          string      `json:"name,omitempty"`
	Url           string      `json:"url,omitempty"`
	Project       TeamProject `json:"project,omitempty"`
	DefaultBranch string      `json:"defaultBranch,omitempty"`
	RemoteUrl     string      `json:"remoteUrl,omitempty"`
	Size          uint64      `json:"size,omitempty"`
	IsFork        bool        `json:"isFork,omitempty"`
}

type TeamProject struct {
	Id   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	Url  string `json:"url,omitempty"`
}

type GitPullRequestCompletionOptions struct {
	BypassPolicy        *bool                        `json:"bypassPolicy,omitempty"`
	BypassReason        *string                      `json:"bypassReason,omitempty"`
	DeleteSourceBranch  *bool                        `json:"deleteSourceBranch,omitempty"`
	MergeCommitMessage  *string                      `json:"mergeCommitMessage,omitempty"`
	MergeStrategy       *GitPullRequestMergeStrategy `json:"mergeStrategy,omitempty"`
	SquashMerge         *bool                        `json:"squashMerge,omitempty"`
	TransitionWorkItems *bool                        `json:"transitionWorkItems,omitempty"`
}

type GitPullRequestMergeOptions struct {
	DetectRenameFalsePositives *bool `json:"detectRenameFalsePositives,omitempty"`
	DisableRenames             *bool `json:"disableRenames,omitempty"`
}

type IdentityRefWithVote struct {
	IdentityRef
	IsRequired *bool `json:"isRequired,omitempty"`
	Vote       *int  `json:"vote,omitempty"`
}

type GitCommitRef struct {
	CommitId string `json:"commitId,omitempty"`
	Comment  string `json:"comment,omitempty"`
	Url      string `json:"url,omitempty"`
}

type GitPullRequest struct {
	Links                 any                              `json:"_links,omitempty"`
	ArtifactId            string                           `json:"artifactId,omitempty"`
	AutoCompleteSetBy     *IdentityRef                     `json:"autoCompleteSetBy,omitempty"`
	ClosedBy              *IdentityRef                     `json:"closedBy,omitempty"`
	ClosedDate            *string                          `json:"closedDate,omitempty"`
	CodeReviewId          *int                             `json:"codeReviewId,omitempty"`
	Commits               *[]GitCommitRef                  `json:"commits,omitempty"`
	CompletionOptions     *GitPullRequestCompletionOptions `json:"completionOptions,omitempty"`
	CompletionQueueTime   *string                          `json:"completionQueueTime,omitempty"`
	CreatedBy             *IdentityRef                     `json:"createdBy,omitempty"`
	CreationDate          *string                          `json:"creationDate,omitempty"`
	Description           string                           `json:"description,omitempty"`
	ForkSource            *GitForkRef                      `json:"forkSource,omitempty"`
	IsDraft               *bool                            `json:"isDraft,omitempty"`
	Labels                *[]any                           `json:"labels,omitempty"`
	LastMergeCommit       *GitCommitRef                    `json:"lastMergeCommit,omitempty"`
	LastMergeSourceCommit *GitCommitRef                    `json:"lastMergeSourceCommit,omitempty"`
	LastMergeTargetCommit *GitCommitRef                    `json:"lastMergeTargetCommit,omitempty"`
	MergeFailureMessage   *string                          `json:"mergeFailureMessage,omitempty"`
	MergeFailureType      *PullRequestMergeFailureType     `json:"mergeFailureType,omitempty"`
	MergeId               *string                          `json:"mergeId,omitempty"`
	MergeOptions          *GitPullRequestMergeOptions      `json:"mergeOptions,omitempty"`
	MergeStatus           *PullRequestAsyncStatus          `json:"mergeStatus,omitempty"`
	PullRequestId         *int                             `json:"pullRequestId,omitempty"`
	RemoteUrl             string                           `json:"remoteUrl,omitempty"`
	Repository            *GitRepository                   `json:"repository,omitempty"`
	Reviewers             *[]IdentityRefWithVote           `json:"reviewers,omitempty"`
	SourceRefName         string                           `json:"sourceRefName,omitempty"`
	Status                *PullRequestStatus               `json:"status,omitempty"`
	SupportsIterations    *bool                            `json:"supportsIterations,omitempty"`
	TargetRefName         string                           `json:"targetRefName,omitempty"`
	Title                 string                           `json:"title,omitempty"`
	Url                   string                           `json:"url,omitempty"`
	WorkItemRefs          *[]any                           `json:"workItemRefs,omitempty"`
}

type PullRequestMergeFailureType string

const (
	PullRequestMergeFailureTypeNone           PullRequestMergeFailureType = "none"
	PullRequestMergeFailureTypeUnknown        PullRequestMergeFailureType = "unknown"
	PullRequestMergeFailureTypeCaseSensitive  PullRequestMergeFailureType = "caseSensitive"
	PullRequestMergeFailureTypeObjectTooLarge PullRequestMergeFailureType = "objectTooLarge"
)

type GitForkRef struct {
	Creator    *IdentityRef   `json:"creator,omitempty"`
	IsLocked   *bool          `json:"isLocked,omitempty"`
	Name       string         `json:"name,omitempty"`
	Repository *GitRepository `json:"repository,omitempty"`
}

type GitPullRequestSearchCriteria struct {
	CreatorId          *string            `json:"creatorId,omitempty"`
	IncludeLinks       *bool              `json:"includeLinks,omitempty"`
	RepositoryId       *string            `json:"repositoryId,omitempty"`
	ReviewerId         *string            `json:"reviewerId,omitempty"`
	SourceRefName      *string            `json:"sourceRefName,omitempty"`
	SourceRepositoryId *string            `json:"sourceRepositoryId,omitempty"`
	Status             *PullRequestStatus `json:"status,omitempty"`
	TargetRefName      *string            `json:"targetRefName,omitempty"`
}

type WorkItemReference struct {
	ID string `json:"id,omitempty"`
}

type BuildStatus string

const (
	BuildStatusNone       BuildStatus = "none"
	BuildStatusInProgress BuildStatus = "inProgress"
	BuildStatusCompleted  BuildStatus = "completed"
	BuildStatusCancelling BuildStatus = "canceling"
	BuildStatusPostponed  BuildStatus = "postponed"
	BuildStatusNotStarted BuildStatus = "notStarted"
	BuildStatusAll        BuildStatus = "all"
)

type BuildResult string

const (
	BuildResultNone               BuildResult = "none"
	BuildResultSucceeded          BuildResult = "succeeded"
	BuildResultPartiallySucceeded BuildResult = "partiallySucceeded"
	BuildResultFailed             BuildResult = "failed"
	BuildResultCanceled           BuildResult = "canceled"
)

type BuildReason string

const (
	BuildReasonNone              BuildReason = "none"
	BuildReasonManual            BuildReason = "manual"
	BuildReasonIndividualCI      BuildReason = "individualCI"
	BuildReasonBatchedCI         BuildReason = "batchedCI"
	BuildReasonSchedule          BuildReason = "schedule"
	BuildReasonScheduleForced    BuildReason = "scheduleForced"
	BuildReasonUserCreated       BuildReason = "userCreated"
	BuildReasonValidateShelveset BuildReason = "validateShelveset"
	BuildReasonCheckInShelveset  BuildReason = "checkInShelveset"
	BuildReasonPullRequest       BuildReason = "pullRequest"
	BuildReasonBuildCompletion   BuildReason = "buildCompletion"
	BuildReasonResourceTrigger   BuildReason = "resourceTrigger"
	BuildReasonTriggered         BuildReason = "triggered"
	BuildReasonAll               BuildReason = "all"
)

type QueuePriority string

const (
	QueuePriorityLow         QueuePriority = "low"
	QueuePriorityBelowNormal QueuePriority = "belowNormal"
	QueuePriorityNormal      QueuePriority = "normal"
	QueuePriorityAboveNormal QueuePriority = "aboveNormal"
	QueuePriorityHigh        QueuePriority = "high"
)

type BuildQueryOrder string

const (
	BuildQueryOrderFinishTimeAscending  BuildQueryOrder = "finishTimeAscending"
	BuildQueryOrderFinishTimeDescending BuildQueryOrder = "finishTimeDescending"
	BuildQueryOrderQueueTimeDescending  BuildQueryOrder = "queueTimeDescending"
	BuildQueryOrderQueueTimeAscending   BuildQueryOrder = "queueTimeAscending"
	BuildQueryOrderStartTimeDescending  BuildQueryOrder = "startTimeDescending"
	BuildQueryOrderStartTimeAscending   BuildQueryOrder = "startTimeAscending"
)

type ReferenceLinks struct {
	Links map[string]any `json:"links,omitempty"`
}

type TeamProjectReference struct {
	Id                  string `json:"id,omitempty"`
	Name                string `json:"name,omitempty"`
	Url                 string `json:"url,omitempty"`
	Abbreviation        string `json:"abbreviation,omitempty"`
	Description         string `json:"description,omitempty"`
	State               string `json:"state,omitempty"`
	Visibility          string `json:"visibility,omitempty"`
	Revision            int64  `json:"revision,omitempty"`
	LastUpdateTime      string `json:"lastUpdateTime,omitempty"`
	DefaultTeamImageUrl string `json:"defaultTeamImageUrl,omitempty"`
}

type DefinitionReference struct {
	Id          int32                 `json:"id,omitempty"`
	Name        string                `json:"name,omitempty"`
	Url         string                `json:"url,omitempty"`
	Path        string                `json:"path,omitempty"`
	Type        string                `json:"type,omitempty"`
	Revision    int32                 `json:"revision,omitempty"`
	Project     *TeamProjectReference `json:"project,omitempty"`
	QueueStatus string                `json:"queueStatus,omitempty"`
	CreatedDate string                `json:"createdDate,omitempty"`
	Uri         string                `json:"uri,omitempty"`
}

type TaskAgentPoolReference struct {
	Id       int32  `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	IsHosted bool   `json:"isHosted,omitempty"`
}

type AgentPoolQueue struct {
	Id    *int32                  `json:"id,omitempty"`
	Name  string                  `json:"name,omitempty"`
	Url   string                  `json:"url,omitempty"`
	Pool  *TaskAgentPoolReference `json:"pool,omitempty"`
	Links *ReferenceLinks         `json:"_links,omitempty"`
}

type BuildRepository struct {
	Id                 string         `json:"id,omitempty"`
	Name               string         `json:"name,omitempty"`
	Url                string         `json:"url,omitempty"`
	Type               string         `json:"type,omitempty"`
	DefaultBranch      string         `json:"defaultBranch,omitempty"`
	Clean              string         `json:"clean,omitempty"`
	CheckoutSubmodules bool           `json:"checkoutSubmodules,omitempty"`
	Properties         map[string]any `json:"properties,omitempty"`
	RootFolder         string         `json:"rootFolder,omitempty"`
}

type BuildLogReference struct {
	Id   int32  `json:"id,omitempty"`
	Type string `json:"type,omitempty"`
	Url  string `json:"url,omitempty"`
}

type TaskOrchestrationPlanReference struct {
	PlanId            string `json:"planId,omitempty"`
	OrchestrationType int32  `json:"orchestrationType,omitempty"`
}

type AgentSpecification struct {
	Identifier string `json:"identifier,omitempty"`
}

type Demand struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

type BuildController struct {
	Id          int32           `json:"id,omitempty"`
	Name        string          `json:"name,omitempty"`
	Url         string          `json:"url,omitempty"`
	Description string          `json:"description,omitempty"`
	Enabled     bool            `json:"enabled,omitempty"`
	Status      string          `json:"status,omitempty"`
	CreatedDate string          `json:"createdDate,omitempty"`
	UpdatedDate string          `json:"updatedDate,omitempty"`
	Uri         string          `json:"uri,omitempty"`
	Links       *ReferenceLinks `json:"_links,omitempty"`
}

type AzureBuild struct {
	Links                        *ReferenceLinks                   `json:"_links,omitempty"`
	Properties                   *PropertiesCollection             `json:"properties,omitempty"`
	Id                           int32                             `json:"id,omitempty"`
	BuildNumber                  string                            `json:"buildNumber,omitempty"`
	BuildNumberRevision          int32                             `json:"buildNumberRevision,omitempty"`
	Status                       BuildStatus                       `json:"status,omitempty"`
	Result                       BuildResult                       `json:"result,omitempty"`
	Reason                       BuildReason                       `json:"reason,omitempty"`
	SourceBranch                 string                            `json:"sourceBranch,omitempty"`
	SourceVersion                string                            `json:"sourceVersion,omitempty"`
	Priority                     QueuePriority                     `json:"priority,omitempty"`
	QueueTime                    string                            `json:"queueTime,omitempty"`
	StartTime                    string                            `json:"startTime,omitempty"`
	FinishTime                   string                            `json:"finishTime,omitempty"`
	QueuePosition                *int32                            `json:"queuePosition,omitempty"`
	Definition                   *DefinitionReference              `json:"definition,omitempty"`
	Project                      *TeamProjectReference             `json:"project,omitempty"`
	Repository                   *BuildRepository                  `json:"repository,omitempty"`
	RequestedBy                  *IdentityRef                      `json:"requestedBy,omitempty"`
	RequestedFor                 *IdentityRef                      `json:"requestedFor,omitempty"`
	LastChangedBy                *IdentityRef                      `json:"lastChangedBy,omitempty"`
	LastChangedDate              string                            `json:"lastChangedDate,omitempty"`
	Parameters                   string                            `json:"parameters,omitempty"`
	OrchestrationPlan            *TaskOrchestrationPlanReference   `json:"orchestrationPlan,omitempty"`
	Logs                         *BuildLogReference                `json:"logs,omitempty"`
	Plans                        []*TaskOrchestrationPlanReference `json:"plans,omitempty"`
	Demands                      []Demand                          `json:"demands,omitempty"`
	Process                      *IdentityRef                      `json:"process,omitempty"`
	Url                          string                            `json:"url,omitempty"`
	Uri                          string                            `json:"uri,omitempty"`
	Deleted                      bool                              `json:"deleted,omitempty"`
	DeletedDate                  string                            `json:"deletedDate,omitempty"`
	DeletedBy                    *IdentityRef                      `json:"deletedBy,omitempty"`
	DeletedReason                string                            `json:"deletedReason,omitempty"`
	KeepForever                  bool                              `json:"keepForever,omitempty"`
	RetainedByRelease            bool                              `json:"retainedByRelease,omitempty"`
	Controller                   *BuildController                  `json:"controller,omitempty"`
	Queue                        *AgentPoolQueue                   `json:"queue,omitempty"`
	AgentSpecification           *AgentSpecification               `json:"agentSpecification,omitempty"`
	BuildNumberFormat            string                            `json:"buildNumberFormat,omitempty"`
	AppendCommitMessageToRunName bool                              `json:"appendCommitMessageToRunName,omitempty"`
	Quality                      string                            `json:"quality,omitempty"`
	Tags                         []string                          `json:"tags,omitempty"`
	ValidationResults            []any                             `json:"validationResults,omitempty"`
	TriggeredByBuild             *AzureBuild                       `json:"triggeredByBuild,omitempty"`
	TemplateParameters           map[string]any                    `json:"templateParameters,omitempty"`
	TriggerInfo                  map[string]any                    `json:"triggerInfo,omitempty"`
	PullRequestId                *int32                            `json:"pullRequestId,omitempty"`
}

type PropertiesCollection struct {
	Count  int            `json:"count,omitempty"`
	Keys   []string       `json:"keys,omitempty"`
	Values []string       `json:"values,omitempty"`
	Item   map[string]any `json:"item,omitempty"`
}

type ProjectReference struct {
	Id   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type DeploymentStatus string

const (
	DeploymentStatusUndefined          DeploymentStatus = "undefined"
	DeploymentStatusNotDeployed        DeploymentStatus = "notDeployed"
	DeploymentStatusInProgress         DeploymentStatus = "inProgress"
	DeploymentStatusSucceeded          DeploymentStatus = "succeeded"
	DeploymentStatusPartiallySucceeded DeploymentStatus = "partiallySucceeded"
	DeploymentStatusFailed             DeploymentStatus = "failed"
	DeploymentStatusAll                DeploymentStatus = "all"
)

type DeploymentOperationStatus string

const (
	DeploymentOperationStatusUndefined                 DeploymentOperationStatus = "undefined"
	DeploymentOperationStatusQueued                    DeploymentOperationStatus = "queued"
	DeploymentOperationStatusScheduled                 DeploymentOperationStatus = "scheduled"
	DeploymentOperationStatusPending                   DeploymentOperationStatus = "pending"
	DeploymentOperationStatusApproved                  DeploymentOperationStatus = "approved"
	DeploymentOperationStatusRejected                  DeploymentOperationStatus = "rejected"
	DeploymentOperationStatusDeferred                  DeploymentOperationStatus = "deferred"
	DeploymentOperationStatusQueuedForAgent            DeploymentOperationStatus = "queuedForAgent"
	DeploymentOperationStatusPhaseInProgress           DeploymentOperationStatus = "phaseInProgress"
	DeploymentOperationStatusPhaseSucceeded            DeploymentOperationStatus = "phaseSucceeded"
	DeploymentOperationStatusPhasePartiallySucceeded   DeploymentOperationStatus = "phasePartiallySucceeded"
	DeploymentOperationStatusPhaseFailed               DeploymentOperationStatus = "phaseFailed"
	DeploymentOperationStatusCanceled                  DeploymentOperationStatus = "canceled"
	DeploymentOperationStatusPhaseCanceled             DeploymentOperationStatus = "phaseCanceled"
	DeploymentOperationStatusManualInterventionPending DeploymentOperationStatus = "manualInterventionPending"
	DeploymentOperationStatusQueuedForPipeline         DeploymentOperationStatus = "queuedForPipeline"
	DeploymentOperationStatusCancelling                DeploymentOperationStatus = "canceling"
	DeploymentOperationStatusEvaluatingGates           DeploymentOperationStatus = "evaluatingGates"
	DeploymentOperationStatusGateFailed                DeploymentOperationStatus = "gateFailed"
	DeploymentOperationStatusAll                       DeploymentOperationStatus = "all"
)

type DeploymentReason string

const (
	DeploymentReasonNone            DeploymentReason = "none"
	DeploymentReasonManual          DeploymentReason = "manual"
	DeploymentReasonAutomated       DeploymentReason = "automated"
	DeploymentReasonScheduled       DeploymentReason = "scheduled"
	DeploymentReasonRedeployTrigger DeploymentReason = "redeployTrigger"
)

type ReleaseShallowReference struct {
	Id        int32           `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Url       string          `json:"url,omitempty"`
	Artifacts []Artifact      `json:"artifacts,omitempty"`
	Links     *ReferenceLinks `json:"_links,omitempty"`
}

type ReleaseDefinitionShallowReference struct {
	Id               int32             `json:"id,omitempty"`
	Name             string            `json:"name,omitempty"`
	Path             string            `json:"path,omitempty"`
	Url              string            `json:"url,omitempty"`
	ProjectReference *ProjectReference `json:"projectReference,omitempty"`
	Links            *ReferenceLinks   `json:"_links,omitempty"`
}

type ReleaseEnvironmentShallowReference struct {
	Id    int32           `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Url   string          `json:"url,omitempty"`
	Links *ReferenceLinks `json:"_links,omitempty"`
}

type Artifact struct {
	Alias               string                             `json:"alias,omitempty"`
	DefinitionReference map[string]ArtifactSourceReference `json:"definitionReference,omitempty"`
	IsPrimary           bool                               `json:"isPrimary,omitempty"`
	IsRetained          bool                               `json:"isRetained,omitempty"`
	SourceId            string                             `json:"sourceId,omitempty"`
	Type                string                             `json:"type,omitempty"`
}

type ArtifactSourceReference struct {
	Id   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type AzureDeployment struct {
	Id                      int32                               `json:"id,omitempty"`
	Release                 *ReleaseShallowReference            `json:"release,omitempty"`
	ReleaseDefinition       *ReleaseDefinitionShallowReference  `json:"releaseDefinition,omitempty"`
	ReleaseEnvironment      *ReleaseEnvironmentShallowReference `json:"releaseEnvironment,omitempty"`
	DefinitionEnvironmentId int32                               `json:"definitionEnvironmentId,omitempty"`
	Attempt                 int32                               `json:"attempt,omitempty"`
	Reason                  DeploymentReason                    `json:"reason,omitempty"`
	DeploymentStatus        DeploymentStatus                    `json:"deploymentStatus,omitempty"`
	OperationStatus         DeploymentOperationStatus           `json:"operationStatus,omitempty"`
	RequestedBy             *IdentityRef                        `json:"requestedBy,omitempty"`
	RequestedFor            *IdentityRef                        `json:"requestedFor,omitempty"`
	QueuedOn                string                              `json:"queuedOn,omitempty"`
	StartedOn               string                              `json:"startedOn,omitempty"`
	CompletedOn             string                              `json:"completedOn,omitempty"`
	LastModifiedOn          string                              `json:"lastModifiedOn,omitempty"`
	LastModifiedBy          *IdentityRef                        `json:"lastModifiedBy,omitempty"`
	Conditions              []Condition                         `json:"conditions,omitempty"`
	ScheduledDeploymentTime string                              `json:"scheduledDeploymentTime,omitempty"`
	PreDeployApprovals      []ReleaseApproval                   `json:"preDeployApprovals,omitempty"`
	PostDeployApprovals     []ReleaseApproval                   `json:"postDeployApprovals,omitempty"`
	Links                   *ReferenceLinks                     `json:"_links,omitempty"`
}

type ReleaseApproval struct {
	Id                 int32                               `json:"id,omitempty"`
	ApprovalType       ApprovalType                        `json:"approvalType,omitempty"`
	Approver           *IdentityRef                        `json:"approver,omitempty"`
	ApprovedBy         *IdentityRef                        `json:"approvedBy,omitempty"`
	Attempt            int32                               `json:"attempt,omitempty"`
	Comments           string                              `json:"comments,omitempty"`
	CreatedOn          string                              `json:"createdOn,omitempty"`
	ModifiedOn         string                              `json:"modifiedOn,omitempty"`
	Rank               int32                               `json:"rank,omitempty"`
	Revision           int32                               `json:"revision,omitempty"`
	Status             ApprovalStatus                      `json:"status,omitempty"`
	IsAutomated        bool                                `json:"isAutomated,omitempty"`
	Release            *ReleaseShallowReference            `json:"release,omitempty"`
	ReleaseDefinition  *ReleaseDefinitionShallowReference  `json:"releaseDefinition,omitempty"`
	ReleaseEnvironment *ReleaseEnvironmentShallowReference `json:"releaseEnvironment,omitempty"`
	Url                string                              `json:"url,omitempty"`
}

type ApprovalType string

const (
	ApprovalTypeUndefined  ApprovalType = "undefined"
	ApprovalTypePreDeploy  ApprovalType = "preDeploy"
	ApprovalTypePostDeploy ApprovalType = "postDeploy"
	ApprovalTypeAll        ApprovalType = "all"
)

type ApprovalStatus string

const (
	ApprovalStatusUndefined  ApprovalStatus = "undefined"
	ApprovalStatusPending    ApprovalStatus = "pending"
	ApprovalStatusApproved   ApprovalStatus = "approved"
	ApprovalStatusRejected   ApprovalStatus = "rejected"
	ApprovalStatusReassigned ApprovalStatus = "reassigned"
	ApprovalStatusCanceled   ApprovalStatus = "canceled"
	ApprovalStatusSkipped    ApprovalStatus = "skipped"
)

type Condition struct {
	ConditionType ConditionType `json:"conditionType,omitempty"`
	Name          string        `json:"name,omitempty"`
	Value         string        `json:"value,omitempty"`
	Result        bool          `json:"result,omitempty"`
}

type ConditionType string

const (
	ConditionTypeUndefined        ConditionType = "undefined"
	ConditionTypeEvent            ConditionType = "event"
	ConditionTypeEnvironmentState ConditionType = "environmentState"
	ConditionTypeArtifact         ConditionType = "artifact"
)

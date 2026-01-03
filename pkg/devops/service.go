package devops

import (
	"fmt"
	"time"

	"adoctl/pkg/azure/client"
	"adoctl/pkg/cache"
	"adoctl/pkg/logger"
)

type SyncOptions struct {
	Quiet    bool
	SkipSync bool
	SyncDone bool
}

type ServiceOptions struct {
	AzureClient *client.Client
	Cache       *cache.Manager
	CachePath   string
}

type DevOpsService struct {
	client      *client.Client
	cache       *cache.Manager
	syncOptions SyncOptions
	buildsSync  bool
}

type BulkCreateResult struct {
	RepoName string
	Success  bool
	PRID     int
	URL      string
	Error    string
}

type ReleaseInfo struct {
	ReleaseID    int
	ReleaseName  string
	Status       string
	Environment  string
	DeploymentID int
	URL          string
	CompletedOn  string
}

type PipelineStatus struct {
	PRID          int
	PRStatus      string
	PRMergeStatus string
	CommitHash    string
	Builds        []BuildStatusInfo
	Deployments   []DeploymentStatusInfo
}

type BuildStatusInfo struct {
	BuildID     int
	BuildNumber string
	Status      string
	Result      string
	Pipeline    string
	URL         string
}

type DeploymentStatusInfo struct {
	ReleaseID       int
	ReleaseName     string
	DeploymentID    int
	Status          string
	Environment     string
	URL             string
	OperationStatus string
	StartTime       time.Time
}

func NewService(opts ServiceOptions) (*DevOpsService, error) {

	var azureClient *client.Client
	var cacheManager *cache.Manager
	var err error

	if opts.AzureClient != nil {
		azureClient = opts.AzureClient
	} else {
		azureClient, err = client.NewClient()
		if err != nil {
			return nil, fmt.Errorf("failed to create Azure client: %w", err)
		}
	}

	if opts.Cache != nil {
		cacheManager = opts.Cache
	} else if opts.CachePath != "" {
		cacheManager, err = cache.NewManager(opts.CachePath)
		if err != nil {
			logger.Warn().Err(err).Msg("Failed to initialize cache")
			cacheManager = nil
		}
	} else {
		cacheManager, err = cache.NewManagerFromEnv()
		if err != nil {
			logger.Warn().Err(err).Msg("Failed to initialize cache")
			cacheManager = nil
		}
	}

	return &DevOpsService{
		client:      azureClient,
		cache:       cacheManager,
		syncOptions: SyncOptions{},
		buildsSync:  false,
	}, nil
}

func NewServiceFromEnv() (*DevOpsService, error) {
	return NewService(ServiceOptions{})
}

func (s *DevOpsService) SetSyncOptions(opts SyncOptions) {
	s.syncOptions = opts
}

func (s *DevOpsService) Close() {
	if s.cache != nil {
		s.cache.Close()
	}
}

func (s *DevOpsService) Client() *client.Client {
	return s.client
}

func (s *DevOpsService) Cache() *cache.Manager {
	return s.cache
}

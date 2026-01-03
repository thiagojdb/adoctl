package cache

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"adoctl/pkg/models"

	_ "github.com/mattn/go-sqlite3"
)

const SELECT_DEPLOYMENT_WHERE = `SELECT 
		release_id, 
		release_name,
		status,
		start_time,
		end_time,
		repository,
		branch,
		source_version,
		build_id,
		full_json,
		updated_at
	FROM deployments WHERE 1=1 
	`
const SELECT_BUILDS_WHERE = `SELECT 
		build_id, 
		branch, 
		repository, 
		source_version, 
		start_time, 
		end_time, 
		status, 
		result, 
		full_json, 
		updated_at
	FROM builds WHERE 1=1 
	`

type CacheConfig struct {
	RepositoriesTTL time.Duration
	UsersTTL        time.Duration
	BuildsTTL       time.Duration
	DeploymentsTTL  time.Duration
}

var DefaultCacheConfig = CacheConfig{
	RepositoriesTTL: 24 * time.Hour,
	UsersTTL:        24 * time.Hour,
	BuildsTTL:       1 * time.Hour,
	DeploymentsTTL:  30 * time.Minute,
}

type Manager struct {
	db     *sql.DB
	config CacheConfig
}

type CachedRepository struct {
	ID          string
	Name        string
	WebURL      string
	ProjectID   string
	ProjectName string
	UpdatedAt   time.Time
}

type User struct {
	ID        string
	Name      string
	UpdatedAt time.Time
}

type Build struct {
	BuildID       int
	Branch        string
	Repository    string
	SourceVersion string
	StartTime     time.Time
	EndTime       sql.NullTime
	Status        string
	Result        string
	FullJSON      string
	UpdatedAt     time.Time
}

type Deployment struct {
	ReleaseID     int
	ReleaseName   string
	Status        string
	StartTime     time.Time
	EndTime       sql.NullTime
	Repository    string
	Branch        string
	SourceVersion string
	BuildID       int
	FullJSON      string
	UpdatedAt     time.Time
}

func NewManager(dbPath string) (*Manager, error) {
	return NewManagerWithConfig(dbPath, DefaultCacheConfig)
}

func NewManagerWithConfig(dbPath string, config CacheConfig) (*Manager, error) {
	if err := os.MkdirAll(dbPath[:len(dbPath)-len("cache.db")], 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	cm := &Manager{db: db, config: config}
	if err := cm.init(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return cm, nil
}

func (cm *Manager) init() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS repositories (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			web_url TEXT,
			project_id TEXT,
			project_name TEXT,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS builds (
			build_id INTEGER PRIMARY KEY,
			branch TEXT NOT NULL,
			repository TEXT NOT NULL,
			source_version TEXT,
			start_time DATETIME NOT NULL,
			end_time DATETIME,
			status TEXT NOT NULL,
			result TEXT NOT NULL,
			full_json TEXT NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS deployments (
			release_id INTEGER PRIMARY KEY,
			release_name TEXT NOT NULL,
			status TEXT NOT NULL,
			start_time DATETIME NOT NULL,
			end_time DATETIME,
			repository TEXT,
			branch TEXT,
			source_version TEXT,
			build_id INTEGER,
			full_json TEXT NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS sync_metadata (
			key TEXT PRIMARY KEY,
			value DATETIME NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_repositories_updated_at ON repositories(updated_at)`,
		`CREATE INDEX IF NOT EXISTS idx_users_updated_at ON users(updated_at)`,
		`CREATE INDEX IF NOT EXISTS idx_builds_status ON builds(status)`,
		`CREATE INDEX IF NOT EXISTS idx_builds_result ON builds(result)`,
		`CREATE INDEX IF NOT EXISTS idx_builds_branch ON builds(branch)`,
		`CREATE INDEX IF NOT EXISTS idx_builds_repository ON builds(repository)`,
		`CREATE INDEX IF NOT EXISTS idx_builds_source_version ON builds(source_version)`,
		`CREATE INDEX IF NOT EXISTS idx_builds_start_time ON builds(start_time)`,
		`CREATE INDEX IF NOT EXISTS idx_deployments_status ON deployments(status)`,
		`CREATE INDEX IF NOT EXISTS idx_deployments_start_time ON deployments(start_time)`,
		`CREATE INDEX IF NOT EXISTS idx_deployments_repository ON deployments(repository)`,
		`CREATE INDEX IF NOT EXISTS idx_deployments_branch ON deployments(branch)`,
	}

	for _, query := range queries {
		if _, err := cm.db.Exec(query); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	return nil
}

func (cm *Manager) Close() error {
	return cm.db.Close()
}

// GetDB returns the underlying database connection
func (cm *Manager) GetDB() *sql.DB {
	return cm.db
}

func (cm *Manager) GetRepositories() ([]models.Repository, error) {
	query := fmt.Sprintf(`SELECT id, name, web_url, project_id, project_name, updated_at
	          FROM repositories
	          WHERE datetime(updated_at) > datetime('now', '-%d seconds')`, int(cm.config.RepositoriesTTL.Seconds()))

	rows, err := cm.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query repositories: %w", err)
	}
	defer rows.Close()

	repos := []models.Repository{}
	for rows.Next() {
		var repo CachedRepository
		var webURL sql.NullString
		var projectID, projectName sql.NullString

		err := rows.Scan(&repo.ID, &repo.Name, &webURL, &projectID, &projectName, &repo.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan repository: %w", err)
		}

		modelRepo := models.Repository{
			ID:        repo.ID,
			Name:      repo.Name,
			URL:       "",
			RemoteURL: "",
		}

		if webURL.Valid {
			modelRepo.URL = webURL.String
		}
		if projectName.Valid {
			modelRepo.Project.Name = projectName.String
		}
		if projectID.Valid {
			modelRepo.Project.ID = projectID.String
		}

		repos = append(repos, modelRepo)
	}

	if len(repos) > 0 {
		return repos, nil
	}

	return nil, nil
}

func (cm *Manager) SetRepositories(repos []models.Repository) error {
	tx, err := cm.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM repositories"); err != nil {
		return fmt.Errorf("failed to clear repositories: %w", err)
	}

	stmt, err := tx.Prepare(`
		INSERT INTO repositories (id, name, web_url, project_id, project_name)
		VALUES (?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, repo := range repos {
		webURL := repo.URL
		projectID := repo.Project.ID
		projectName := repo.Project.Name

		if _, err := stmt.Exec(repo.ID, repo.Name, webURL, projectID, projectName); err != nil {
			return fmt.Errorf("failed to insert repository: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (cm *Manager) GetUsers() (map[string]User, error) {
	query := fmt.Sprintf(`SELECT id, name, updated_at 
	          FROM users 
	          WHERE datetime(updated_at) > datetime('now', '-%d seconds')`, int(cm.config.UsersTTL.Seconds()))

	rows, err := cm.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	users := make(map[string]User)
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Name, &user.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users[user.ID] = user
	}

	if len(users) > 0 {
		return users, nil
	}

	return nil, nil
}

func (cm *Manager) SetUsers(users map[string]map[string]any) error {
	tx, err := cm.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM users"); err != nil {
		return fmt.Errorf("failed to clear users: %w", err)
	}

	stmt, err := tx.Prepare(`
		INSERT INTO users (id, name)
		VALUES (?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for id, user := range users {
		name, _ := user["name"].(string)
		if _, err := stmt.Exec(id, name); err != nil {
			return fmt.Errorf("failed to insert user: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (cm *Manager) GetCacheInfo() (map[string]any, error) {
	var repoCount, userCount int
	var oldestRepo, oldestUser sql.NullTime

	if err := cm.db.QueryRow("SELECT COUNT(*) FROM repositories").Scan(&repoCount); err != nil {
		return nil, fmt.Errorf("failed to query repository count: %w", err)
	}
	if err := cm.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount); err != nil {
		return nil, fmt.Errorf("failed to query user count: %w", err)
	}
	if err := cm.db.QueryRow("SELECT MIN(updated_at) FROM repositories").Scan(&oldestRepo); err != nil {
		return nil, fmt.Errorf("failed to query oldest repository: %w", err)
	}
	if err := cm.db.QueryRow("SELECT MIN(updated_at) FROM users").Scan(&oldestUser); err != nil {
		return nil, fmt.Errorf("failed to query oldest user: %w", err)
	}

	return map[string]any{
		"repositories_count": repoCount,
		"users_count":        userCount,
		"oldest_repo":        oldestRepo.Time,
		"oldest_user":        oldestUser.Time,
	}, nil
}

func GetDBPath() string {
	homeDir, err := os.UserCacheDir()
	if err != nil {
		homeDir = os.TempDir()
	}
	return fmt.Sprintf("%s/adoctl/cache.db", homeDir)
}

func NewManagerFromEnv() (*Manager, error) {
	config := LoadCacheConfig()
	return NewManagerWithConfig(GetDBPath(), config)
}

func LoadCacheConfig() CacheConfig {
	config := DefaultCacheConfig

	if ttlStr := os.Getenv("ADOCTL_CACHE_REPOS_TTL"); ttlStr != "" {
		if d, err := time.ParseDuration(ttlStr); err == nil {
			config.RepositoriesTTL = d
		}
	}

	if ttlStr := os.Getenv("ADOCTL_CACHE_USERS_TTL"); ttlStr != "" {
		if d, err := time.ParseDuration(ttlStr); err == nil {
			config.UsersTTL = d
		}
	}

	if ttlStr := os.Getenv("ADOCTL_CACHE_BUILDS_TTL"); ttlStr != "" {
		if d, err := time.ParseDuration(ttlStr); err == nil {
			config.BuildsTTL = d
		}
	}

	if ttlStr := os.Getenv("ADOCTL_CACHE_DEPLOYMENTS_TTL"); ttlStr != "" {
		if d, err := time.ParseDuration(ttlStr); err == nil {
			config.DeploymentsTTL = d
		}
	}

	return config
}

func (cm *Manager) SaveBuild(build Build) error {
	query := `
		INSERT OR REPLACE INTO builds 
		(build_id, branch, repository, source_version, start_time, end_time, status, result, full_json, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`

	var endTime any
	if build.EndTime.Valid {
		endTime = build.EndTime.Time
	} else {
		endTime = nil
	}

	_, err := cm.db.Exec(query, build.BuildID, build.Branch, build.Repository, build.SourceVersion, build.StartTime, endTime, build.Status, build.Result, build.FullJSON)
	if err != nil {
		return fmt.Errorf("failed to save build: %w", err)
	}

	return nil
}

func (cm *Manager) GetBuildByID(buildID int) (*Build, error) {
	query := `SELECT build_id, branch, repository, source_version, start_time, end_time, status, full_json, updated_at
	          FROM builds WHERE build_id = ?`

	row := cm.db.QueryRow(query, buildID)

	var build Build
	var endTime sql.NullTime
	err := row.Scan(&build.BuildID, &build.Branch, &build.Repository, &build.SourceVersion, &build.StartTime, &endTime, &build.Status, &build.FullJSON, &build.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get build: %w", err)
	}

	build.EndTime = endTime
	return &build, nil
}

func (cm *Manager) SearchBuilds(filters map[string]any) ([]Build, error) {
	query := SELECT_BUILDS_WHERE
	args := []any{}
	argIndex := 1

	if buildID, ok := filters["build_id"].(int); ok {
		query += " AND build_id = ?"
		args = append(args, buildID)
		argIndex++
	}

	if branch, ok := filters["branch"].(string); ok && branch != "" {
		query += " AND branch = ?"
		args = append(args, branch)
		argIndex++
	}

	if repository, ok := filters["repository"].(string); ok && repository != "" {
		query += " AND repository = ?"
		args = append(args, repository)
		argIndex++
	}

	if commit, ok := filters["commit"].(string); ok && commit != "" {
		query += " AND source_version LIKE ?"
		args = append(args, commit+"%")
		argIndex++
	}

	if status, ok := filters["status"].(string); ok && status != "" {
		query += " AND status = ?"
		args = append(args, status)
		argIndex++
	}

	if startTimeFrom, ok := filters["start_time_from"].(time.Time); ok {
		query += " AND start_time >= ?"
		args = append(args, startTimeFrom)
		argIndex++
	}

	if startTimeTo, ok := filters["start_time_to"].(time.Time); ok {
		query += " AND start_time <= ?"
		args = append(args, startTimeTo)
		argIndex++
	}

	if endTimeFrom, ok := filters["end_time_from"].(time.Time); ok {
		query += " AND end_time >= ?"
		args = append(args, endTimeFrom)
		argIndex++
	}

	if endTimeTo, ok := filters["end_time_to"].(time.Time); ok {
		query += " AND end_time <= ?"
		args = append(args, endTimeTo)
		argIndex++
	}

	if hasEndTime, ok := filters["has_end_time"].(bool); ok {
		if hasEndTime {
			query += " AND end_time IS NOT NULL"
		} else {
			query += " AND end_time IS NULL"
		}
		argIndex++
	}

	query += " ORDER BY start_time DESC"

	if limit, ok := filters["limit"].(int); ok && limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := cm.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search builds: %w", err)
	}
	defer rows.Close()

	builds := []Build{}
	for rows.Next() {
		var build Build
		err := build.Scan(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan build: %w", err)
		}
		builds = append(builds, build)
	}

	return builds, nil
}

func (cm *Manager) GetLastSyncTime(key string) (*time.Time, error) {
	query := `SELECT value FROM sync_metadata WHERE key = ?`

	var syncTime time.Time
	err := cm.db.QueryRow(query, key).Scan(&syncTime)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get last sync time: %w", err)
	}

	return &syncTime, nil
}

func (cm *Manager) SetLastSyncTime(key string, syncTime time.Time) error {
	query := `
		INSERT OR REPLACE INTO sync_metadata (key, value)
		VALUES (?, ?)
	`

	_, err := cm.db.Exec(query, key, syncTime)
	if err != nil {
		return fmt.Errorf("failed to set last sync time: %w", err)
	}

	return nil
}

func (cm *Manager) GetAllBuilds() ([]Build, error) {
	query := SELECT_BUILDS_WHERE

	query += `ORDER BY start_time DESC`

	rows, err := cm.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query all builds: %w", err)
	}
	defer rows.Close()

	builds := []Build{}
	for rows.Next() {
		var build Build
		var endTime sql.NullTime

		err := rows.Scan(&build.BuildID, &build.Branch, &build.Repository, &build.SourceVersion, &build.StartTime, &endTime, &build.Status, &build.FullJSON, &build.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan build: %w", err)
		}

		build.EndTime = endTime
		builds = append(builds, build)
	}

	return builds, nil
}

func (cm *Manager) SaveDeployment(deployment Deployment) error {
	query := `
		INSERT OR REPLACE INTO deployments
		(release_id, release_name, status, start_time, end_time, repository, branch, source_version, build_id, full_json, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`

	var endTime any
	if deployment.EndTime.Valid {
		endTime = deployment.EndTime.Time
	}

	_, err := cm.db.Exec(query,
		deployment.ReleaseID,
		deployment.ReleaseName,
		deployment.Status,
		deployment.StartTime,
		endTime,
		deployment.Repository,
		deployment.Branch,
		deployment.SourceVersion,
		deployment.BuildID,
		deployment.FullJSON)
	if err != nil {
		return fmt.Errorf("failed to save deployment: %w", err)
	}

	return nil
}

func (cm *Manager) SearchDeployments(filters map[string]any) ([]Deployment, error) {
	query := SELECT_DEPLOYMENT_WHERE
	args := []any{}
	argIndex := 1

	if releaseID, ok := filters["release_id"].(int); ok {
		query += " AND release_id = ?"
		args = append(args, releaseID)
		argIndex++
	}

	if buildID, ok := filters["build_id"].(int); ok {
		query += " AND build_id = ?"
		args = append(args, buildID)
		argIndex++
	}

	if releaseName, ok := filters["release_name"].(string); ok && releaseName != "" {
		query += " AND release_name LIKE ?"
		args = append(args, "%"+releaseName+"%")
		argIndex++
	}

	if status, ok := filters["status"].(string); ok && status != "" {
		query += " AND status = ?"
		args = append(args, status)
		argIndex++
	}

	if repository, ok := filters["repository"].(string); ok && repository != "" {
		query += " AND repository = ?"
		args = append(args, repository)
		argIndex++
	}

	if branch, ok := filters["branch"].(string); ok && branch != "" {
		query += " AND branch = ?"
		args = append(args, branch)
		argIndex++
	}

	if startTimeFrom, ok := filters["start_time_from"].(time.Time); ok {
		query += " AND start_time >= ?"
		args = append(args, startTimeFrom)
		argIndex++
	}

	if startTimeTo, ok := filters["start_time_to"].(time.Time); ok {
		query += " AND start_time <= ?"
		args = append(args, startTimeTo)
		argIndex++
	}

	if endTimeFrom, ok := filters["end_time_from"].(time.Time); ok {
		query += " AND end_time >= ?"
		args = append(args, endTimeFrom)
		argIndex++
	}

	if endTimeTo, ok := filters["end_time_to"].(time.Time); ok {
		query += " AND end_time <= ?"
		args = append(args, endTimeTo)
		argIndex++
	}

	if hasEndTime, ok := filters["has_end_time"].(bool); ok {
		if hasEndTime {
			query += " AND end_time IS NOT NULL"
		} else {
			query += " AND end_time IS NULL"
		}
		argIndex++
	}

	query += " ORDER BY start_time DESC"

	if limit, ok := filters["limit"].(int); ok && limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := cm.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search deployments: %w", err)
	}
	defer rows.Close()

	deployments := []Deployment{}
	for rows.Next() {
		var deployment Deployment
		err := ScanRows(rows, &deployment)
		if err != nil {
			return nil, fmt.Errorf("failed to scan deployment: %w", err)
		}

		deployments = append(deployments, deployment)
	}

	return deployments, nil
}

func ScanRow(row *sql.Row, deployment *Deployment) error {
	return row.Scan(&deployment.ReleaseID,
		&deployment.ReleaseName,
		&deployment.Status,
		&deployment.StartTime,
		&deployment.EndTime,
		&deployment.Repository,
		&deployment.Branch,
		&deployment.SourceVersion,
		&deployment.BuildID,
		&deployment.FullJSON,
		&deployment.UpdatedAt)
}

func ScanRows(rows *sql.Rows, deployment *Deployment) error {
	return rows.Scan(&deployment.ReleaseID,
		&deployment.ReleaseName,
		&deployment.Status,
		&deployment.StartTime,
		&deployment.EndTime,
		&deployment.Repository,
		&deployment.Branch,
		&deployment.SourceVersion,
		&deployment.BuildID,
		&deployment.FullJSON,
		&deployment.UpdatedAt)
}

func (build *Build) Scan(rows *sql.Rows) error {
	return rows.Scan(&build.BuildID,
		&build.Branch,
		&build.Repository,
		&build.SourceVersion,
		&build.StartTime,
		&build.EndTime,
		&build.Status,
		&build.Result,
		&build.FullJSON,
		&build.UpdatedAt)
}

func (cm *Manager) GetAllDeployments() ([]Deployment, error) {
	query := SELECT_DEPLOYMENT_WHERE
	query += `ORDER BY start_time DESC`

	rows, err := cm.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query all deployments: %w", err)
	}
	defer rows.Close()

	deployments := []Deployment{}
	for rows.Next() {
		var deployment Deployment
		err := ScanRows(rows, &deployment)
		if err != nil {
			return nil, fmt.Errorf("failed to scan deployment: %w", err)
		}

		deployments = append(deployments, deployment)
	}

	return deployments, nil
}

package groupapi

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/fatih/structs"
	"github.com/razorpay/trino-gateway/internal/boot"
	"github.com/razorpay/trino-gateway/internal/gatewayserver/metrics"
	"github.com/razorpay/trino-gateway/internal/gatewayserver/models"
	"github.com/razorpay/trino-gateway/internal/gatewayserver/repo"
	"github.com/razorpay/trino-gateway/internal/provider"
	"github.com/razorpay/trino-gateway/internal/utils"
)

type Core struct {
	groupRepo   repo.IGroupRepo
	backendRepo repo.IBackendRepo
}

type ICore interface {
	CreateOrUpdateGroup(ctx context.Context, params *GroupCreateParams) error
	GetGroup(ctx context.Context, id string) (*models.Group, error)
	GetAllGroups(ctx context.Context) ([]models.Group, error)
	GetAllActiveGroups(ctx context.Context) ([]models.Group, error)
	DeleteGroup(ctx context.Context, id string) error
	EnableGroup(ctx context.Context, id string) error
	DisableGroup(ctx context.Context, id string) error

	EvaluateBackendForGroups(ctx context.Context, groups []string) (string, string, error)
}

func NewCore(group repo.IGroupRepo, backend repo.IBackendRepo) *Core {
	return &Core{groupRepo: group, backendRepo: backend}
}

// CreateParams has attributes that are required for group.Create()
type GroupCreateParams struct {
	ID                string
	Strategy          string
	IsEnabled         bool
	LastRoutedBackend string
	Backends          []string
}

func (c *Core) CreateOrUpdateGroup(ctx context.Context, params *GroupCreateParams) error {
	var backendMappings []models.GroupBackendsMapping
	for _, backend := range params.Backends {
		backendMappings = append(backendMappings, models.GroupBackendsMapping{
			GroupId:   params.ID,
			BackendId: backend,
		})
	}

	group := models.Group{
		Strategy:          &params.Strategy,
		IsEnabled:         &params.IsEnabled,
		LastRoutedBackend: &params.LastRoutedBackend,
	}
	group.ID = params.ID
	group.GroupBackendsMappings = backendMappings
	_, notexists := c.groupRepo.Find(ctx, params.ID)
	if notexists == nil { // update
		return c.groupRepo.Update(ctx, &group)
	} else { // create
		return c.groupRepo.Create(ctx, &group)
	}
}

func (c *Core) GetGroup(ctx context.Context, id string) (*models.Group, error) {
	group, err := c.groupRepo.Find(ctx, id)
	return group, err
}

func (c *Core) GetAllGroups(ctx context.Context) ([]models.Group, error) {
	groups, err := c.groupRepo.FindMany(ctx, make(map[string]interface{}))
	return groups, err
}

type IFindManyParams interface {
	// GetCount() int32
	// GetSkip() int32
	// GetFrom() int32
	// GetTo() int32

	// custom
	GetIsEnabled() bool
}

type FindManyParams struct {
	// pagination
	// Count int32
	// Skip  int32
	// From  int32
	// To    int32

	// custom
	IsEnabled bool `json:"is_enabled"`
}

func (p *FindManyParams) GetIsEnabled() bool {
	return p.IsEnabled
}

func (c *Core) FindMany(ctx context.Context, params IFindManyParams) ([]models.Group, error) {
	conditionStr := structs.New(params)
	// use the json tag name, so we can respect omitempty tags
	conditionStr.TagName = "json"
	conditions := conditionStr.Map()

	return c.groupRepo.FindMany(ctx, conditions)
}

func (c *Core) GetAllActiveGroups(ctx context.Context) ([]models.Group, error) {
	groups, err := c.FindMany(ctx, &FindManyParams{IsEnabled: true})
	return groups, err
}

func (c *Core) DeleteGroup(ctx context.Context, id string) error {
	return c.groupRepo.Delete(ctx, id)
}

func (c *Core) EnableGroup(ctx context.Context, id string) error {
	return c.groupRepo.Enable(ctx, id)
}

func (c *Core) DisableGroup(ctx context.Context, id string) error {
	return c.groupRepo.Disable(ctx, id)
}

func (c *Core) EvaluateBackendForGroups(ctx context.Context, groups []string) (string, string, error) {
	// Step 1: Get all active groups
	provider.Logger(ctx).Debug("Fetching all active groups")

	var chosenGroup *models.Group
	activeGroups, err := c.GetAllActiveGroups(ctx)
	if err != nil {
		return "", "", err
	}

	// Step 2: take intersections of all non nil grp sets; a nil set = any grp; all sets nil == route to fallbackGrp;
	provider.Logger(ctx).Debug("Take intersection of active grps with provided grp list")
	var eligibleGrps []*models.Group
	for i, g := range activeGroups {
		if utils.SliceContains(groups, g.ID) {
			eligibleGrps = append(eligibleGrps, &activeGroups[i])
		}
	}

	// Step 3: Evaluate Backends for each
	evaluatedBackendId := make(map[*models.Group]string, len(eligibleGrps))
	for _, g := range eligibleGrps {
		backend_id, err := c.findBackend(ctx, *g)
		if err != nil {
			return "", "", err
		}
		if backend_id != nil {
			evaluatedBackendId[g] = *backend_id
		}
	}

	// Step 4: Choose group & a backend
	var chosenBackendId string
	if len(evaluatedBackendId) > 0 {
		chosenGroup = eligibleGrps[0]
		chosenBackendId = evaluatedBackendId[chosenGroup]

	} else {
		// fallback grp
		provider.Logger(ctx).Logger.Warn(
			"No eligible backends available, invoking fallback group routing.",
		)
		metrics.FallbackGroupInvoked.WithLabelValues().Inc()
		chosenGroup, err = c.GetGroup(ctx, boot.Config.Gateway.DefaultRoutingGroup)
		if err != nil {
			return "", "", err
		}
		b, err := c.findBackend(ctx, *chosenGroup)
		if err != nil {
			return "", "", err
		}
		if b == nil {
			return "", "", errors.New("unable to find Backend for Default Routing Group")
		}
		chosenBackendId = *b
	}

	provider.Logger(ctx).Debugw("Backend Evaluated for groups", map[string]interface{}{
		"chosenGroupId":              chosenGroup.GetID(),
		"chosenBackendId":            chosenBackendId,
		"evaluatedBackendsForGroups": fmt.Sprint(evaluatedBackendId),
	})

	return chosenBackendId, chosenGroup.GetID(), nil
}

func (c *Core) findBackend(ctx context.Context, group models.Group) (*string, error) {
	provider.Logger(ctx).Infow("Choose a backend for group", map[string]interface{}{"group": group.GetID()})

	// Step 0: get all backends
	// var backends := (*group.GroupBackendsMappings)[0].BackendId

	provider.Logger(ctx).Debugw("Fetch all backends in the group", map[string]interface{}{"group": group.GetID()})
	backends := make([]string, len(group.GroupBackendsMappings))
	for i, k := range group.GroupBackendsMappings {
		backends[i] = k.BackendId
	}

	provider.Logger(ctx).Debugw("Filter active backends", map[string]interface{}{"group": group.GetID(), "backends": backends})
	// Step 1: Filter Active backends
	activeBackends, err := c.backendRepo.GetAllActiveByIDs(ctx, backends)
	if err != nil {
		return nil, err
	}

	if len(activeBackends) == 0 {
		return nil, nil
	}

	// Step 2: Evaluate strategy
	selectedBackendId := (activeBackends)[0].ID
	provider.Logger(ctx).Debugw("Evaluate strategy for the group", map[string]interface{}{"group": group.GetID(), "strategy": *group.Strategy})
	switch strategy := *group.Strategy; strategy {
	case "round_robin":
		activeBackendIds := make([]string, len(activeBackends))
		for i, b := range activeBackends {
			activeBackendIds[i] = b.GetID()
		}
		lastRoutedBackendId := *group.LastRoutedBackend
		activeBackendIds = append(activeBackendIds, lastRoutedBackendId)
		sort.Strings(activeBackendIds)

		index := 0
		for i, b := range activeBackendIds {
			if b == lastRoutedBackendId {
				index = i
			}
		}
		index = index + 1
		if index >= len(activeBackendIds) {
			index = 0
		}
		selectedBackendId = activeBackendIds[index]

	case "least_load":
		leastLoaded := (activeBackends)[0]
		load := func(b *models.Backend) int {
			curr := time.Now().Unix()
			validityS := boot.Config.Monitor.StatsValiditySecs
			if validityS == 0 || curr-*b.StatsUpdatedAt <= int64(validityS) {
				provider.Logger(ctx).Debugw(
					"ClusterLoad stats in valid time range",
					map[string]interface{}{"backend": b.GetID(), "cluster_load": *b.ClusterLoad},
				)
				return int(*b.ClusterLoad)
			}
			provider.Logger(ctx).Infow(
				"ClusterLoad stats too old to be valid, assuming load as 0",
				map[string]interface{}{
					"backend":          b.GetID(),
					"validity_period":  validityS,
					"stats_updated_at": *b.StatsUpdatedAt,
				})
			return 0
		}
		provider.Logger(ctx).Info("Selecting least loaded backend")
		for _, b := range activeBackends {
			if load(&b) < load(&leastLoaded) {
				leastLoaded = b
			}
		}
		selectedBackendId = leastLoaded.ID
	default:
		provider.Logger(ctx).Debugw("Falling back to `random` strategy for group", map[string]interface{}{"group": group.GetID(), "strategy": *group.Strategy})
	}
	// case RANDOM: return any
	// case ROUND_ROBIN: order by ascending and take next bck_id after last_routed_backend
	// case LOAD_BASED: get metrics of each backend and choose one with lowest running+queued_queries
	provider.Logger(ctx).Debugw("Backend evaluated for group", map[string]interface{}{"group": group.GetID(), "strategy": *group.Strategy, "backend": selectedBackendId})

	updGrp := models.Group{LastRoutedBackend: &selectedBackendId}
	updGrp.ID = group.ID
	c.groupRepo.Update(ctx, &updGrp)

	return &selectedBackendId, nil
}

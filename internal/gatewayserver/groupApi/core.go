package groupapi

import (
	"context"
	"errors"
	"sort"

	"github.com/fatih/structs"
	"github.com/razorpay/trino-gateway/internal/boot"
	"github.com/razorpay/trino-gateway/internal/gatewayserver/models"
	"github.com/razorpay/trino-gateway/internal/gatewayserver/repo"
)

type Core struct {
	groupRepo   repo.IGroupRepo
	backendRepo repo.IBackendRepo
}

type ICore interface {
	CreateOrUpdateGroup(ctx context.Context, params *GroupCreateParams) error
	GetGroup(ctx context.Context, id string) (*models.Group, error)
	GetAllGroups(ctx context.Context) (*[]models.Group, error)
	GetAllActiveGroups(ctx context.Context) (*[]models.Group, error)
	DeleteGroup(ctx context.Context, id string) error
	EnableGroup(ctx context.Context, id string) error
	DisableGroup(ctx context.Context, id string) error

	EvaluateBackendForGroups(ctx context.Context, groups []string) (string, string, error)
}

func NewCore(ctx *context.Context, group repo.IGroupRepo, backend repo.IBackendRepo) *Core {
	_ = ctx
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
	group.GroupBackendsMappings = &backendMappings
	_, exists := c.groupRepo.Find(ctx, params.ID)
	if exists == nil { // update
		return c.groupRepo.Update(ctx, &group)
	} else { // create
		return c.groupRepo.Create(ctx, &group)
	}
}

func (c *Core) GetGroup(ctx context.Context, id string) (*models.Group, error) {
	group, err := c.groupRepo.Find(ctx, id)
	return group, err
}

func (c *Core) GetAllGroups(ctx context.Context) (*[]models.Group, error) {
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
	Count int32
	Skip  int32
	From  int32
	To    int32

	// custom
	IsEnabled bool
}

func (p *FindManyParams) GetIsEnabled() bool {
	return p.IsEnabled
}

func (c *Core) FindMany(ctx context.Context, params IFindManyParams) (*[]models.Group, error) {

	conditionStr := structs.New(params)
	// use the json tag name, so we can respect omitempty tags
	conditionStr.TagName = "json"
	conditions := conditionStr.Map()

	return c.groupRepo.FindMany(ctx, conditions)
}

func (c *Core) GetAllActiveGroups(ctx context.Context) (*[]models.Group, error) {
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
	// TODO

	// Step 2: take intersections of all non nil grp sets; a nil set = any grp; all sets nil == route to fallbackGrp;
	// Step 3: find active grps
	var chosenGroup *models.Group
	activeGroups, err := c.GetAllActiveGroups(ctx)
	if err != nil {
		return "", "", err
	}

	// Evaluate Backend for each
	evaluatedBackendId := make(map[models.Group]*string, len(*activeGroups))
	for _, g := range *activeGroups {
		backend_id, err := c.findBackend(ctx, g)
		if err != nil {
			return "", "", err
		}
		evaluatedBackendId[g] = backend_id
		if backend_id != nil {
			chosenGroup = &g
		}

	}

	var chosenBackendId string
	if chosenGroup != nil {
		chosenBackendId = *evaluatedBackendId[*chosenGroup]
	} else {
		chosenGroup, err = c.GetGroup(ctx, boot.Config.Gateway.DefaultRoutingGroup)
		if err != nil {
			return "", "", err
		}
		b, err := c.findBackend(ctx, *chosenGroup)
		if err != nil {
			return "", "", err
		}
		if b == nil {
			return "", "", errors.New("Unable to find Backend for Default Routing Group")
		}
	}

	return chosenBackendId, chosenGroup.GetID(), nil
}

func (c *Core) findBackend(ctx context.Context, group models.Group) (*string, error) {

	// Step 0: get all backends
	//var backends := (*group.GroupBackendsMappings)[0].BackendId

	backends := make([]string, 0, len(*group.GroupBackendsMappings))
	for i, k := range *group.GroupBackendsMappings {
		backends[i] = k.BackendId
	}

	// Step 1: Filter Active backends
	activeBackends, err := c.backendRepo.FindMany(ctx, map[string]interface{}{
		"is_enabled": true,
		"id in (?)":  backends,
	})
	if err != nil {
		return nil, err
	}

	if len(*activeBackends) == 0 {
		return nil, nil
	}

	// Step 2: Evaluate strategy
	selectedBackendId := (*activeBackends)[0].ID
	switch strategy := *group.Strategy; strategy {
	case "ROUND_ROBIN":
		activeBackendIds := make([]string, 0, len(*activeBackends))
		for i, b := range *activeBackends {
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
		if index+1 >= len(activeBackendIds) {
			index = 0
		}
		selectedBackendId = activeBackendIds[index]

	case "LEAST_LOADED":
		leastLoaded := (*activeBackends)[0]
		load := func(b models.Backend) int {
			return int(*b.RunningQueries) + int(*b.QueuedQueries)
		}
		for _, b := range *activeBackends {
			if load(b) < load(leastLoaded) {
				leastLoaded = b
			}
		}
		selectedBackendId = leastLoaded.ID
	default:
	}
	// case RANDOM: return any
	// case ROUND_ROBIN: order by ascending and take next bck_id after last_routed_backend
	// case LOAD_BASED: get metrics of each backend and choose one with lowest running+queued_queries
	return &selectedBackendId, nil
}
